package config

import "strings"

func normalizeProviders(providers []ProviderConfig) []ProviderConfig {
	seen := make(map[string]struct{}, len(providers))
	result := make([]ProviderConfig, 0, len(providers))
	for _, provider := range providers {
		provider.ID = strings.ToLower(strings.TrimSpace(provider.ID))
		provider.Name = strings.TrimSpace(provider.Name)
		provider.APIType = strings.ToLower(strings.TrimSpace(provider.APIType))
		provider.BaseURL = strings.TrimSpace(provider.BaseURL)
		if provider.ID == "" {
			continue
		}
		if provider.Name == "" {
			provider.Name = fallbackProviderName(provider.ID)
		}
		if provider.APIType == "" {
			provider.APIType = "openai-compatible"
		}
		if _, exists := seen[provider.ID]; exists {
			continue
		}
		seen[provider.ID] = struct{}{}
		result = append(result, provider)
	}
	return result
}

func fallbackProviderName(id string) string {
	switch id {
	case "zhipu":
		return "Zhipu"
	case "openai":
		return "OpenAI"
	default:
		return id
	}
}
