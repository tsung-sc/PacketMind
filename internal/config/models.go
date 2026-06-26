package config

import "strings"

type ModelLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

type ModelModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type ModelConfig struct {
	Name       string           `json:"name"`
	Limit      *ModelLimit      `json:"limit,omitempty"`
	Modalities *ModelModalities `json:"modalities,omitempty"`
}

type ProviderConfig struct {
	ID      string                  `json:"id"`
	Name    string                  `json:"name,omitempty"`
	APIType string                  `json:"api_type"`
	APIKey  string                  `json:"api_key,omitempty"`
	BaseURL string                  `json:"base_url,omitempty"`
	Models  map[string]ModelConfig  `json:"models,omitempty"`
}

type ModelsConfig struct {
	Providers         []ProviderConfig `json:"providers"`
	DefaultModel      string           `json:"default_model,omitempty"`
	ActiveProvider    string           `json:"active_provider,omitempty"`
	ActiveModel       string           `json:"active_model,omitempty"`
	ActiveMaxTokens   int              `json:"active_max_tokens,omitempty"`
	ActiveTemperature float64          `json:"active_temperature,omitempty"`
}

type ProviderInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	APIType    string `json:"api_type"`
	Icon       string `json:"icon,omitempty"`
	HasAPIKey  bool   `json:"has_api_key"`
	HasBaseURL bool   `json:"has_base_url"`
	ModelCount int    `json:"model_count"`
	IsActive   bool   `json:"is_active"`
	Deletable  bool   `json:"deletable"`
}

type ModelsByProvider struct {
	Provider     string        `json:"provider"`
	ProviderName string        `json:"provider_name"`
	ProviderIcon string        `json:"provider_icon,omitempty"`
	Models       []ModelConfig `json:"models"`
}

type AgentSettings struct {
	Provider    string  `json:"provider"`
	APIType     string  `json:"api_type"`
	APIKey      string  `json:"api_key,omitempty"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type UpdateAgentSettings struct {
	Provider    *string  `json:"provider"`
	APIType     *string  `json:"api_type"`
	APIKey      *string  `json:"api_key"`
	BaseURL     *string  `json:"base_url"`
	Model       *string  `json:"model"`
	MaxTokens   *int     `json:"max_tokens"`
	Temperature *float64 `json:"temperature"`
}

func (m *ModelsConfig) GetModelByID(id string) *ModelConfig {
	if m == nil {
		return nil
	}
	id = strings.TrimSpace(id)
	for i := range m.Providers {
		if m.Providers[i].Models != nil {
			if cfg, ok := m.Providers[i].Models[id]; ok {
				return &cfg
			}
		}
	}
	return nil
}

func (m *ModelsConfig) GetModelByProviderAndID(provider, id string) *ModelConfig {
	if m == nil {
		return nil
	}
	id = strings.TrimSpace(id)
	if p := m.GetProviderByID(provider); p != nil && p.Models != nil {
		if cfg, ok := p.Models[id]; ok {
			return &cfg
		}
	}
	return nil
}

func (m *ModelsConfig) GetModelsByProvider(provider string) []ModelConfig {
	if m == nil {
		return nil
	}
	p := m.GetProviderByID(provider)
	if p == nil || p.Models == nil {
		return nil
	}
	result := make([]ModelConfig, 0, len(p.Models))
	for _, model := range p.Models {
		result = append(result, model)
	}
	return result
}

func (m *ModelsConfig) allModelsCount() int {
	if m == nil {
		return 0
	}
	count := 0
	for i := range m.Providers {
		count += len(m.Providers[i].Models)
	}
	return count
}

func (m *ModelsConfig) firstModelID() string {
	if m == nil {
		return ""
	}
	for i := range m.Providers {
		for id := range m.Providers[i].Models {
			return id
		}
	}
	return ""
}

func (m *ModelsConfig) GetDefaultModel() *ModelConfig {
	return m.GetModelByID(m.DefaultModel)
}

func (m *ModelsConfig) GetAPIKey(provider string) string {
	if cfg := m.GetProviderByID(provider); cfg != nil {
		return cfg.APIKey
	}
	return ""
}

func (m *ModelsConfig) GetBaseURL(provider string) string {
	if cfg := m.GetProviderByID(provider); cfg != nil {
		return cfg.BaseURL
	}
	return ""
}

func (m *ModelsConfig) GetAPIType(provider string) string {
	if cfg := m.GetProviderByID(provider); cfg != nil {
		return cfg.APIType
	}
	return ""
}

func (m *ModelsConfig) GetProviderByID(provider string) *ProviderConfig {
	provider = strings.TrimSpace(strings.ToLower(provider))
	for i := range m.Providers {
		if strings.EqualFold(m.Providers[i].ID, provider) {
			copy := m.Providers[i]
			return &copy
		}
	}
	return nil
}

func (m *ModelsConfig) currentAISettingsLocked() AgentSettings {
	settings := AgentSettings{
		Provider:    m.ActiveProvider,
		APIType:     m.GetAPIType(m.ActiveProvider),
		Model:       m.ActiveModel,
		MaxTokens:   m.ActiveMaxTokens,
		Temperature: m.ActiveTemperature,
	}

	if settings.Model == "" {
		settings.Model = m.DefaultModel
	}
	if settings.Provider == "" || m.GetProviderByID(settings.Provider) == nil {
		settings.Provider = m.firstProviderID()
	}
	if settings.APIType == "" {
		settings.APIType = m.GetAPIType(settings.Provider)
	}
	if settings.APIType == "" {
		settings.APIType = "openai-compatible"
	}
	if settings.Model == "" {
		settings.Model = m.DefaultModel
	}

	if settings.MaxTokens <= 0 {
		if model := m.GetModelByID(settings.Model); model != nil && model.Limit != nil && model.Limit.Output > 0 {
			settings.MaxTokens = model.Limit.Output
		} else {
			settings.MaxTokens = 2000
		}
	}

	if settings.Temperature == 0 {
		settings.Temperature = 0.7
	}

	settings.BaseURL = m.GetBaseURL(settings.Provider)
	settings.APIKey = m.GetAPIKey(settings.Provider)
	return settings
}

func (m *ModelsConfig) normalize() {
	m.Providers = normalizeProviders(m.Providers)
	for i := range m.Providers {
		if m.Providers[i].Models == nil {
			m.Providers[i].Models = make(map[string]ModelConfig)
		}
	}
	if m.DefaultModel == "" {
		m.DefaultModel = m.firstModelID()
	}
	if m.ActiveModel == "" {
		m.ActiveModel = m.DefaultModel
	}
	if m.ActiveProvider == "" || m.GetProviderByID(m.ActiveProvider) == nil {
		m.ActiveProvider = m.firstProviderID()
	}
	if m.ActiveMaxTokens <= 0 {
		if model := m.GetModelByID(m.ActiveModel); model != nil && model.Limit != nil && model.Limit.Output > 0 {
			m.ActiveMaxTokens = model.Limit.Output
		} else {
			m.ActiveMaxTokens = 2000
		}
	}
}

func (m *ModelsConfig) firstProviderID() string {
	if m == nil || len(m.Providers) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(m.Providers[0].ID))
}

func (m *ModelsConfig) setProvider(provider ProviderConfig) {
	provider.ID = strings.ToLower(strings.TrimSpace(provider.ID))
	for i := range m.Providers {
		if strings.EqualFold(m.Providers[i].ID, provider.ID) {
			m.Providers[i] = provider
			return
		}
	}
	m.Providers = append(m.Providers, provider)
}

func (m *ModelsConfig) deleteProvider(providerID string) bool {
	for i := range m.Providers {
		if strings.EqualFold(m.Providers[i].ID, providerID) {
			m.Providers = append(m.Providers[:i], m.Providers[i+1:]...)
			return true
		}
	}
	return false
}
