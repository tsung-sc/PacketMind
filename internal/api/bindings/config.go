package bindings

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"

	"github.com/packetmind/packetmind/internal/config"
	"github.com/packetmind/packetmind/internal/mcpserver"
	"github.com/packetmind/packetmind/internal/proxy"
	"github.com/packetmind/packetmind/internal/storage"
)

// ConfigAPI 提供配置相关的前端绑定。
type ConfigAPI struct{}

// NewConfigAPI 创建 ConfigAPI 实例。
func NewConfigAPI() *ConfigAPI {
	return &ConfigAPI{}
}

// GetConfig 获取脱敏后的运行配置。
func (c *ConfigAPI) GetConfig() SessionResponse {
	agentSettings := config.DefaultModelsStore.GetSettings()
	appSettings := config.DefaultSettingsStore.Snapshot()
	safeConfig := map[string]interface{}{
		"agent": map[string]interface{}{
			"provider":    agentSettings.Provider,
			"api_type":    agentSettings.APIType,
			"model":       agentSettings.Model,
			"max_tokens":  agentSettings.MaxTokens,
			"temperature": agentSettings.Temperature,
			"base_url":    agentSettings.BaseURL,
		},
		"desktop": sanitizeSettings(appSettings),
	}

	return SessionResponse{Code: 0, Data: safeConfig}
}

// GetSettings 获取应用设置。
func (c *ConfigAPI) GetSettings() SessionResponse {
	return SessionResponse{Code: 0, Data: sanitizeSettings(config.DefaultSettingsStore.Snapshot())}
}

type UpdateSettingsRequest = config.AppSettings

// UpdateSettings 更新应用设置并同步应用到代理运行时。
func (c *ConfigAPI) UpdateSettings(req UpdateSettingsRequest) SessionResponse {
	oldSettings := config.DefaultSettingsStore.Snapshot()
	nextSettings := config.ResolveRuntimeSettings((*config.AppSettings)(&req))

	oldRuntimeSettings := config.ResolveRuntimeSettings(oldSettings)
	if err := proxy.Default.ApplySettings(nextSettings); err != nil {
		return SessionResponse{Code: 40002, Message: fmt.Sprintf("Failed to apply settings: %v", err)}
	}

	settings, err := config.DefaultSettingsStore.Update(config.AppSettings(req))
	if err != nil {
		if rollbackErr := proxy.Default.ApplySettings(oldRuntimeSettings); rollbackErr != nil {
			fmt.Printf("[ConfigAPI] failed to rollback proxy settings: %v\n", rollbackErr)
		}
		return SessionResponse{Code: 40002, Message: err.Error()}
	}

	if oldSettings.MCPServer.Enabled != settings.MCPServer.Enabled ||
		oldSettings.MCPServer.Host != settings.MCPServer.Host ||
		oldSettings.MCPServer.Port != settings.MCPServer.Port {
		_ = mcpserver.StopSSEServer()
		if settings.MCPServer.Enabled {
			if err := mcpserver.StartSSEServer(storage.Default, settings.MCPServer.Host, settings.MCPServer.Port); err != nil {
				fmt.Printf("[ConfigAPI] failed to start MCP server: %v\n", err)
			}
		}
	}

	return SessionResponse{Code: 0, Message: "Settings updated", Data: sanitizeSettings(settings)}
}

// UpdateAgentConfigRequest 描述 Agent 配置更新请求。
type UpdateAgentConfigRequest struct {
	Provider    *string  `json:"provider"`
	APIType     *string  `json:"api_type"`
	APIKey      *string  `json:"api_key"`
	BaseURL     *string  `json:"base_url"`
	Model       *string  `json:"model"`
	MaxTokens   *int     `json:"max_tokens"`
	Temperature *float64 `json:"temperature"`
}

type UpsertAgentProviderRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	APIType string `json:"api_type"`
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

type DeleteAgentProviderRequest struct {
	ID string `json:"id"`
}

type UpsertModelRequest struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Context  int    `json:"context"`
	Output   int    `json:"output"`
}

type DeleteModelRequest struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

// UpdateAgentConfig 更新 Agent 运行时配置。
func (c *ConfigAPI) UpdateAgentConfig(req UpdateAgentConfigRequest) SessionResponse {
	update := config.UpdateAgentSettings{}
	if req.Provider != nil {
		update.Provider = req.Provider
	}
	if req.APIType != nil {
		update.APIType = req.APIType
	}
	if req.APIKey != nil {
		update.APIKey = req.APIKey
	}
	if req.BaseURL != nil {
		update.BaseURL = req.BaseURL
	}
	if req.Model != nil {
		update.Model = req.Model
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		update.MaxTokens = req.MaxTokens
	}
	if req.Temperature != nil && *req.Temperature >= 0 {
		update.Temperature = req.Temperature
	}

	settings, err := config.DefaultModelsStore.UpdateSettings(update)
	if err != nil {
		return SessionResponse{Code: 40002, Message: err.Error()}
	}

	return SessionResponse{
		Code:    0,
		Message: "Agent config updated",
		Data: map[string]interface{}{
			"provider":    settings.Provider,
			"api_type":    settings.APIType,
			"model":       settings.Model,
			"max_tokens":  settings.MaxTokens,
			"temperature": settings.Temperature,
			"base_url":    settings.BaseURL,
		},
	}
}

func (c *ConfigAPI) UpsertAgentProvider(req UpsertAgentProviderRequest) SessionResponse {
	provider, err := config.DefaultModelsStore.UpsertProvider(config.ProviderConfig{
		ID:      req.ID,
		Name:    req.Name,
		APIType: req.APIType,
		APIKey:  req.APIKey,
		BaseURL: req.BaseURL,
	})
	if err != nil {
		return SessionResponse{Code: 40002, Message: err.Error()}
	}
	return SessionResponse{Code: 0, Data: provider, Message: "provider updated"}
}

func (c *ConfigAPI) DeleteAgentProvider(req DeleteAgentProviderRequest) SessionResponse {
	if err := config.DefaultModelsStore.DeleteProvider(req.ID); err != nil {
		return SessionResponse{Code: 40002, Message: err.Error()}
	}
	return SessionResponse{Code: 0, Message: "provider deleted"}
}

func (c *ConfigAPI) UpsertModel(req UpsertModelRequest) SessionResponse {
	if err := config.DefaultModelsStore.UpsertModel(req.Provider, req.ID, config.ModelConfig{
		Name: req.Name,
		Limit: &config.ModelLimit{
			Context: req.Context,
			Output:  req.Output,
		},
	}); err != nil {
		return SessionResponse{Code: 40002, Message: err.Error()}
	}
	return SessionResponse{Code: 0, Message: "model saved"}
}

func (c *ConfigAPI) DeleteModel(req DeleteModelRequest) SessionResponse {
	if err := config.DefaultModelsStore.DeleteModel(req.Provider, req.ID); err != nil {
		return SessionResponse{Code: 40002, Message: err.Error()}
	}
	return SessionResponse{Code: 0, Message: "model deleted"}
}

func (c *ConfigAPI) GetAgentProviderKey(provider string) SessionResponse {
	providerConfig := config.DefaultModelsStore.GetProvider(provider)
	if providerConfig == nil {
		return SessionResponse{Code: 0, Data: map[string]interface{}{
			"api_key":  "",
			"base_url": "",
			"api_type": "",
		}}
	}
	return SessionResponse{Code: 0, Data: map[string]interface{}{
		"api_key":  providerConfig.APIKey,
		"base_url": providerConfig.BaseURL,
		"api_type": providerConfig.APIType,
	}}
}

// GetCertInfo 获取 CA 证书文件状态信息。
func (c *ConfigAPI) GetCertInfo() SessionResponse {
	certPath := proxy.Default.GetCACertPath()
	info := FileInfoDTO{
		Path:     certPath,
		Exists:   false,
		Download: "/api/config/cert",
	}

	if stat, err := os.Stat(certPath); err == nil {
		info.Exists = true
		info.Size = stat.Size()
		info.Modified = toJSONTime(stat.ModTime())
	}

	return SessionResponse{Code: 0, Data: info}
}

// DownloadCert 下载 CA 证书并返回 Base64 内容。
func (c *ConfigAPI) DownloadCert() SessionResponse {
	certPath := proxy.Default.GetCACertPath()
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return SessionResponse{
			Code:    40001,
			Message: "CA certificate not found. Start proxy first to generate certificate.",
		}
	}

	data, err := os.ReadFile(certPath)
	if err != nil {
		return SessionResponse{Code: 50001, Message: "Failed to read certificate"}
	}

	return SessionResponse{
		Code: 0,
		Data: map[string]interface{}{
			"filename": "packetmind-ca.crt",
			"content":  base64.StdEncoding.EncodeToString(data),
		},
	}
}

func (c *ConfigAPI) GetLocalIPAddresses() SessionResponse {
	interfaces, err := net.Interfaces()
	if err != nil {
		return SessionResponse{Code: 50001, Message: fmt.Sprintf("failed to list network interfaces: %v", err)}
	}

	items := make([]LocalIPAddressDTO, 0)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			default:
				continue
			}

			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}

			items = append(items, LocalIPAddressDTO{
				InterfaceName: iface.Name,
				IPAddress:     ip.String(),
			})
		}
	}

	return SessionResponse{Code: 0, Data: items}
}

func sanitizeSettings(settings *config.AppSettings) map[string]interface{} {
	if settings == nil {
		settings = &config.AppSettings{}
	}
	return map[string]interface{}{
		"cert": map[string]interface{}{
			"ca_cert":      settings.Cert.CACertFile,
			"ca_key":       settings.Cert.CAKeyFile,
			"organization": settings.Cert.Organization,
		},
		"proxy": map[string]interface{}{
			"listener": map[string]interface{}{
				"http_enabled":       settings.Proxy.Listener.HTTPEnabled,
				"port":               settings.Proxy.Listener.Port,
				"https_enabled":      settings.Proxy.Listener.HTTPSEnabled,
				"mitm_enabled":       settings.Proxy.Listener.MITMEnabled,
				"socks5_enabled":     settings.Proxy.Listener.SOCKS5Enabled,
				"auto_start_on_boot": settings.Proxy.Listener.AutoStartOnBoot,
			},
			"recording": map[string]interface{}{
				"enabled":                  settings.Proxy.Recording.Enabled,
				"max_capture_body_size_mb": settings.Proxy.Recording.MaxCaptureBodySizeMB,
			},
			"ssl_proxying": map[string]interface{}{
				"enabled":       settings.Proxy.SSLProxying.Enabled,
				"include_hosts": settings.Proxy.SSLProxying.IncludeHosts,
				"exclude_hosts": settings.Proxy.SSLProxying.ExcludeHosts,
			},
			"access_control": map[string]interface{}{
				"enabled":         settings.Proxy.AccessControl.Enabled,
				"allowed_clients": settings.Proxy.AccessControl.AllowedClients,
			},
			"external_proxy": map[string]interface{}{
				"enabled":             settings.Proxy.ExternalProxy.Enabled,
				"scheme":              settings.Proxy.ExternalProxy.Scheme,
				"host":                settings.Proxy.ExternalProxy.Host,
				"port":                settings.Proxy.ExternalProxy.Port,
				"username":            settings.Proxy.ExternalProxy.Username,
				"password_configured": settings.Proxy.ExternalProxy.Password != "",
				"bypass_hosts":        settings.Proxy.ExternalProxy.BypassHosts,
			},
			"throttling": map[string]interface{}{
				"enabled":         settings.Proxy.Throttling.Enabled,
				"latency_ms":      settings.Proxy.Throttling.LatencyMs,
				"downstream_kbps": settings.Proxy.Throttling.DownstreamKBPS,
				"upstream_kbps":   settings.Proxy.Throttling.UpstreamKBPS,
			},
			"breakpoints": map[string]interface{}{
				"enabled":           settings.Proxy.Breakpoints.Enabled,
				"request_matchers":  settings.Proxy.Breakpoints.RequestMatchers,
				"response_matchers": settings.Proxy.Breakpoints.ResponseMatchers,
			},
			"reverse_proxy": map[string]interface{}{
				"enabled": settings.Proxy.ReverseProxy.Enabled,
				"rules":   settings.Proxy.ReverseProxy.Rules,
			},
			"port_forwarding": map[string]interface{}{
				"enabled": settings.Proxy.PortForwarding.Enabled,
				"rules":   settings.Proxy.PortForwarding.Rules,
			},
			"web_interface": map[string]interface{}{
				"enabled": settings.Proxy.WebInterface.Enabled,
				"port":    settings.Proxy.WebInterface.Port,
			},
		},
		"tools": map[string]interface{}{
			"no_caching":         settings.Tools.NoCaching,
			"block_cookies":      settings.Tools.BlockCookies,
			"map_remote_enabled": settings.Tools.MapRemoteEnabled,
			"map_local_enabled":  settings.Tools.MapLocalEnabled,
			"rewrite_enabled":    settings.Tools.RewriteEnabled,
			"block_list_enabled": settings.Tools.BlockListEnabled,
			"dns_spoofing":       settings.Tools.DNSSpoofing,
			"mirror_enabled":     settings.Tools.MirrorEnabled,
			"auto_save_enabled":  settings.Tools.AutoSaveEnabled,
			"client_process":     settings.Tools.ClientProcess,
		},
		"window": map[string]interface{}{
			"structure_view": settings.Window.StructureView,
			"use_dark_theme": settings.Window.UseDarkTheme,
		},
		"mcp_server": map[string]interface{}{
			"enabled": settings.MCPServer.Enabled,
			"host":    settings.MCPServer.Host,
			"port":    settings.MCPServer.Port,
		},
	}
}
