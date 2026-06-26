package bindings

import (
	"context"

	"github.com/packetmind/packetmind/internal/proxy"
)

// ProxyAPI 提供代理控制相关的前端绑定。
type ProxyAPI struct{}

// NewProxyAPI 创建 ProxyAPI 实例。
func NewProxyAPI() *ProxyAPI {
	return &ProxyAPI{}
}

// ProxyStatus 描述当前代理运行状态。
type ProxyStatus struct {
	Running bool `json:"running"`
}

// Status 获取代理状态。
func (p *ProxyAPI) Status() SessionResponse {
	return SessionResponse{
		Code: 0,
		Data: ProxyStatus{Running: proxy.Default.IsRunning()},
	}
}

// Start 启动代理。
func (p *ProxyAPI) Start() SessionResponse {
	if err := proxy.Default.Start(context.Background()); err != nil {
		return SessionResponse{Code: 50003, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Message: "Proxy started"}
}

// Stop 停止代理。
func (p *ProxyAPI) Stop() SessionResponse {
	if err := proxy.Default.Stop(); err != nil {
		return SessionResponse{Code: 50003, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Message: "Proxy stopped"}
}
