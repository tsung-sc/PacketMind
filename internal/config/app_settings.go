package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var DefaultDataDir string

// AppSettings represents the desktop/proxy configuration.
type AppSettings struct {
	Proxy     ProxySettings     `json:"proxy"`
	Tools     ToolSettings      `json:"tools"`
	Window    WindowSettings    `json:"window"`
	Cert      CertSettings      `json:"cert"`
	MCP       MCPSettings       `json:"mcp"`
	MCPServer MCPServerSettings `json:"mcp_server"`
}

// CertSettings holds CA certificate configuration.
type CertSettings struct {
	CACertFile   string `json:"ca_cert"`
	CAKeyFile    string `json:"ca_key"`
	Organization string `json:"organization"`
}

// ProxySettings holds all proxy-related configuration.
type ProxySettings struct {
	Listener       ProxyListenerSettings  `json:"listener"`
	Recording      RecordingSettings      `json:"recording"`
	SSLProxying    SSLProxyingSettings    `json:"ssl_proxying"`
	AccessControl  AccessControlSettings  `json:"access_control"`
	ExternalProxy  ExternalProxySettings  `json:"external_proxy"`
	Throttling     ThrottlingSettings     `json:"throttling"`
	Breakpoints    BreakpointSettings     `json:"breakpoints"`
	ReverseProxy   ReverseProxySettings   `json:"reverse_proxy"`
	PortForwarding PortForwardingSettings `json:"port_forwarding"`
	WebInterface   WebInterfaceSettings   `json:"web_interface"`
}

// ProxyListenerSettings configures the shared proxy listener.
type ProxyListenerSettings struct {
	HTTPEnabled     bool `json:"http_enabled"`
	Port            int  `json:"port"`
	HTTPSEnabled    bool `json:"https_enabled"`
	MITMEnabled     bool `json:"mitm_enabled"`
	SOCKS5Enabled   bool `json:"socks5_enabled"`
	AutoStartOnBoot bool `json:"auto_start_on_boot"`
}

// RecordingSettings configures traffic recording behavior.
type RecordingSettings struct {
	Enabled              bool `json:"enabled"`
	MaxCaptureBodySizeMB int  `json:"max_capture_body_size_mb"`
}

// SSLProxyingSettings configures HTTPS MITM decryption.
type SSLProxyingSettings struct {
	Enabled      bool     `json:"enabled"`
	IncludeHosts []string `json:"include_hosts"`
	ExcludeHosts []string `json:"exclude_hosts"`
}

// AccessControlSettings configures client IP access control.
type AccessControlSettings struct {
	Enabled        bool     `json:"enabled"`
	AllowedClients []string `json:"allowed_clients"`
}

// ExternalProxySettings configures upstream proxy chaining.
type ExternalProxySettings struct {
	Enabled     bool     `json:"enabled"`
	Scheme      string   `json:"scheme"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Username    string   `json:"username"`
	Password    string   `json:"password,omitempty"`
	BypassHosts []string `json:"bypass_hosts"`
}

// ThrottlingSettings configures bandwidth throttling.
type ThrottlingSettings struct {
	Enabled        bool `json:"enabled"`
	LatencyMs      int  `json:"latency_ms"`
	DownstreamKBPS int  `json:"downstream_kbps"`
	UpstreamKBPS   int  `json:"upstream_kbps"`
}

// BreakpointSettings configures request/response breakpoints.
type BreakpointSettings struct {
	Enabled          bool     `json:"enabled"`
	RequestMatchers  []string `json:"request_matchers"`
	ResponseMatchers []string `json:"response_matchers"`
}

// ReverseProxyRule defines a single reverse proxy mapping.
type ReverseProxyRule struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// ReverseProxySettings configures reverse proxy rules.
type ReverseProxySettings struct {
	Enabled bool               `json:"enabled"`
	Rules   []ReverseProxyRule `json:"rules"`
}

// PortForwardRule defines a single port forwarding entry.
type PortForwardRule struct {
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port"`
}

// PortForwardingSettings configures port forwarding rules.
type PortForwardingSettings struct {
	Enabled bool              `json:"enabled"`
	Rules   []PortForwardRule `json:"rules"`
}

// WebInterfaceSettings configures the web interface.
type WebInterfaceSettings struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// ToolSettings configures proxy tool features.
type ToolSettings struct {
	NoCaching        bool `json:"no_caching"`
	BlockCookies     bool `json:"block_cookies"`
	MapRemoteEnabled bool `json:"map_remote_enabled"`
	MapLocalEnabled  bool `json:"map_local_enabled"`
	RewriteEnabled   bool `json:"rewrite_enabled"`
	BlockListEnabled bool `json:"block_list_enabled"`
	DNSSpoofing      bool `json:"dns_spoofing"`
	MirrorEnabled    bool `json:"mirror_enabled"`
	AutoSaveEnabled  bool `json:"auto_save_enabled"`
	ClientProcess    bool `json:"client_process"`
}

// WindowSettings configures desktop window preferences.
type WindowSettings struct {
	StructureView bool `json:"structure_view"`
	UseDarkTheme  bool `json:"use_dark_theme"`
}

type MCPServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool              `json:"enabled"`
}

type MCPSettings struct {
	Servers []MCPServerConfig `json:"servers"`
}

type MCPServerSettings struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

// DefaultPacketMindSettings returns the default desktop/proxy settings.
func DefaultPacketMindSettings() *AppSettings {
	settings := &AppSettings{
		Proxy: ProxySettings{
			Listener: ProxyListenerSettings{
				HTTPEnabled:   true,
				Port:          8888,
				HTTPSEnabled:  true,
				MITMEnabled:   true,
				SOCKS5Enabled: true,
			},
			Recording: RecordingSettings{
				Enabled:              true,
				MaxCaptureBodySizeMB: 5,
			},
			ExternalProxy: ExternalProxySettings{
				Scheme:      "http",
				BypassHosts: []string{},
			},
			SSLProxying: SSLProxyingSettings{
				IncludeHosts: []string{},
				ExcludeHosts: []string{},
			},
			AccessControl: AccessControlSettings{
				AllowedClients: []string{},
			},
			Breakpoints: BreakpointSettings{
				RequestMatchers:  []string{},
				ResponseMatchers: []string{},
			},
			ReverseProxy: ReverseProxySettings{
				Rules: []ReverseProxyRule{},
			},
			PortForwarding: PortForwardingSettings{
				Rules: []PortForwardRule{},
			},
			WebInterface: WebInterfaceSettings{
				Port: 8889,
			},
		},
		Window: WindowSettings{
			StructureView: true,
		},
		Cert: CertSettings{
			CACertFile:   "./data/certs/ca.crt",
			CAKeyFile:    "./data/certs/ca.key",
			Organization: "PacketMind",
		},
		MCPServer: MCPServerSettings{
			Enabled: false,
			Host:    "127.0.0.1",
			Port:    8889,
		},
	}
	settings.normalize()
	return settings
}

// LoadPacketMindSettings loads desktop/proxy settings from packetmind.json.
func LoadPacketMindSettings(configPath string) (*AppSettings, error) {
	path := packetMindSettingsPath(configPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultPacketMindSettings(), nil
		}
		return nil, err
	}

	settings := DefaultPacketMindSettings()
	if err := json.Unmarshal(data, settings); err != nil {
		return nil, err
	}
	settings.normalize()
	return settings, nil
}

// SavePacketMindSettings persists desktop/proxy settings with atomic write.
func SavePacketMindSettings(path string, settings *AppSettings) error {
	if settings == nil {
		return fmt.Errorf("packetmind settings are nil")
	}
	target := packetMindSettingsPath(path)
	payload := cloneAppSettings(settings)
	payload.normalize()

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal packetmind settings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("failed to create packetmind settings dir: %w", err)
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp packetmind settings: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to replace packetmind settings: %w", err)
	}
	return nil
}

// CloneForRuntime returns a normalized deep copy of settings for runtime use.
func ResolveRuntimeSettings(settings *AppSettings) *AppSettings {
	cloned := CloneForRuntime(settings)
	if cloned == nil || strings.TrimSpace(DefaultDataDir) == "" {
		return cloned
	}
	cloned.Cert.CACertFile = resolveDataPath(DefaultDataDir, cloned.Cert.CACertFile)
	cloned.Cert.CAKeyFile = resolveDataPath(DefaultDataDir, cloned.Cert.CAKeyFile)
	return cloned
}

func resolveDataPath(dataDir, path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return trimmed
	}
	cleaned := filepath.Clean(trimmed)
	dataPrefix := "data" + string(os.PathSeparator)
	if strings.HasPrefix(cleaned, dataPrefix) {
		cleaned = strings.TrimPrefix(cleaned, dataPrefix)
	}
	return filepath.Join(dataDir, cleaned)
}

func CloneForRuntime(settings *AppSettings) *AppSettings {
	cloned := cloneAppSettings(settings)
	cloned.normalize()
	return cloned
}

func (d *AppSettings) normalize() {

	wasZero := isAppSettingsZeroValue(*d)

	if d.Proxy.Listener.Port <= 0 {
		d.Proxy.Listener.Port = 8888
	}
	if wasZero {
		d.Proxy.Listener.HTTPEnabled = true
		d.Proxy.Listener.HTTPSEnabled = true
		d.Proxy.Listener.MITMEnabled = true
		d.Proxy.Listener.SOCKS5Enabled = true
	}

	if d.Proxy.Recording.MaxCaptureBodySizeMB <= 0 {
		d.Proxy.Recording.MaxCaptureBodySizeMB = 5
	}
	if !d.Proxy.Recording.Enabled && wasZero {
		d.Proxy.Recording.Enabled = true
	}
	if d.Proxy.ExternalProxy.Scheme == "" {
		d.Proxy.ExternalProxy.Scheme = "http"
	}
	if d.Proxy.ExternalProxy.BypassHosts == nil {
		d.Proxy.ExternalProxy.BypassHosts = []string{}
	}
	if d.Proxy.Throttling.DownstreamKBPS < 0 {
		d.Proxy.Throttling.DownstreamKBPS = 0
	}
	if d.Proxy.Throttling.UpstreamKBPS < 0 {
		d.Proxy.Throttling.UpstreamKBPS = 0
	}
	if d.Proxy.Throttling.LatencyMs < 0 {
		d.Proxy.Throttling.LatencyMs = 0
	}
	if d.Proxy.WebInterface.Port <= 0 {
		d.Proxy.WebInterface.Port = 8889
	}
	if wasZero {
		d.Window.StructureView = true
	}
	if d.Cert.CACertFile == "" {
		d.Cert.CACertFile = "./data/certs/ca.crt"
	}
	if d.Cert.CAKeyFile == "" {
		d.Cert.CAKeyFile = "./data/certs/ca.key"
	}
	if d.Cert.Organization == "" {
		d.Cert.Organization = "PacketMind"
	}
	if d.Proxy.SSLProxying.IncludeHosts == nil {
		d.Proxy.SSLProxying.IncludeHosts = []string{}
	}
	if d.Proxy.SSLProxying.ExcludeHosts == nil {
		d.Proxy.SSLProxying.ExcludeHosts = []string{}
	}
	if d.Proxy.AccessControl.AllowedClients == nil {
		d.Proxy.AccessControl.AllowedClients = []string{}
	}
	if d.Proxy.Breakpoints.RequestMatchers == nil {
		d.Proxy.Breakpoints.RequestMatchers = []string{}
	}
	if d.Proxy.Breakpoints.ResponseMatchers == nil {
		d.Proxy.Breakpoints.ResponseMatchers = []string{}
	}
	if d.Proxy.ReverseProxy.Rules == nil {
		d.Proxy.ReverseProxy.Rules = []ReverseProxyRule{}
	}
	if d.Proxy.PortForwarding.Rules == nil {
		d.Proxy.PortForwarding.Rules = []PortForwardRule{}
	}
}

func isAppSettingsZeroValue(settings AppSettings) bool {
	return settings.Proxy.Listener == (ProxyListenerSettings{}) &&
		settings.Proxy.Recording == (RecordingSettings{}) &&
		!settings.Proxy.SSLProxying.Enabled &&
		len(settings.Proxy.SSLProxying.IncludeHosts) == 0 &&
		len(settings.Proxy.SSLProxying.ExcludeHosts) == 0 &&
		!settings.Proxy.AccessControl.Enabled &&
		len(settings.Proxy.AccessControl.AllowedClients) == 0 &&
		!settings.Proxy.ExternalProxy.Enabled &&
		settings.Proxy.ExternalProxy.Scheme == "" &&
		settings.Proxy.ExternalProxy.Host == "" &&
		settings.Proxy.ExternalProxy.Port == 0 &&
		settings.Proxy.ExternalProxy.Username == "" &&
		settings.Proxy.ExternalProxy.Password == "" &&
		len(settings.Proxy.ExternalProxy.BypassHosts) == 0 &&
		settings.Proxy.Throttling == (ThrottlingSettings{}) &&
		!settings.Proxy.Breakpoints.Enabled &&
		len(settings.Proxy.Breakpoints.RequestMatchers) == 0 &&
		len(settings.Proxy.Breakpoints.ResponseMatchers) == 0 &&
		len(settings.Proxy.ReverseProxy.Rules) == 0 &&
		len(settings.Proxy.PortForwarding.Rules) == 0 &&
		settings.Proxy.WebInterface == (WebInterfaceSettings{}) &&
		settings.Tools == (ToolSettings{}) &&
		settings.Window == (WindowSettings{}) &&
		settings.Cert == (CertSettings{})
}

func cloneAppSettings(settings *AppSettings) *AppSettings {
	if settings == nil {
		return &AppSettings{}
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return &AppSettings{}
	}
	var cloned AppSettings
	if err := json.Unmarshal(data, &cloned); err != nil {
		return &AppSettings{}
	}
	return &cloned
}

func packetMindSettingsPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "packetmind.json"
	}
	if strings.EqualFold(filepath.Base(trimmed), "packetmind.json") {
		return trimmed
	}
	return filepath.Join(trimmed, "packetmind.json")
}
