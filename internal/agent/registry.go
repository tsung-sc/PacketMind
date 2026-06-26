package agent

import (
	"fmt"
	"strings"
	"sync"

	openaipkg "github.com/packetmind/packetmind/internal/agent/provider/openai"
)

type ProviderFactory struct {
	NewAgent func(apiKey, baseURL string) (*Agent, error)
}

type ProviderDescriptor struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Factory     ProviderFactory
}

type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]ProviderDescriptor
	order     []string
}

var defaultRegistry = &ProviderRegistry{
	providers: make(map[string]ProviderDescriptor),
}

func init() {
	RegisterProvider(ProviderDescriptor{
		ID:   "openai-compatible",
		Name: "OpenAI-Compatible",
		Icon: "🇨🇳",
		Factory: ProviderFactory{
			NewAgent: func(apiKey, baseURL string) (*Agent, error) {
				return NewAgent(openaipkg.NewClient(apiKey, baseURL, openaipkg.Config{
					ProviderName: "OpenAI-Compatible",
					ProviderID:   "openai-compatible",
				})), nil
			},
		},
	})
}

func RegisterProvider(desc ProviderDescriptor) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	desc.ID = strings.TrimSpace(desc.ID)
	if _, exists := defaultRegistry.providers[desc.ID]; !exists {
		defaultRegistry.order = append(defaultRegistry.order, desc.ID)
	}
	defaultRegistry.providers[desc.ID] = desc
}

func GetProviderFactory(provider string) (ProviderFactory, error) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	provider = strings.ToLower(strings.TrimSpace(provider))
	desc, ok := defaultRegistry.providers[provider]
	if !ok {
		return ProviderFactory{}, fmt.Errorf("unsupported provider: %s", provider)
	}
	return desc.Factory, nil
}

func GetProviderDescriptor(provider string) (ProviderDescriptor, bool) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	provider = strings.ToLower(strings.TrimSpace(provider))
	desc, ok := defaultRegistry.providers[provider]
	return desc, ok
}
