package config

import (
	"strings"
	"testing"
)

func TestDefaultPacketMindSettings_AppliesExpectedDefaults(t *testing.T) {
	settings := DefaultPacketMindSettings()
	if settings.Cert.CACertFile != "./data/certs/ca.crt" || settings.Cert.CAKeyFile != "./data/certs/ca.key" || settings.Cert.Organization != "PacketMind" {
		t.Fatalf("unexpected cert defaults: %+v", settings.Cert)
	}
	if settings.Proxy.Listener.Port != 8888 || !settings.Proxy.Listener.HTTPEnabled || !settings.Proxy.Listener.HTTPSEnabled || !settings.Proxy.Listener.MITMEnabled || !settings.Proxy.Listener.SOCKS5Enabled {
		t.Fatalf("unexpected listener defaults: %+v", settings.Proxy.Listener)
	}
	if !settings.Proxy.Recording.Enabled || settings.Proxy.Recording.MaxCaptureBodySizeMB != 5 {
		t.Fatalf("unexpected recording defaults: %+v", settings.Proxy.Recording)
	}
	if settings.Proxy.ExternalProxy.Scheme != "http" {
		t.Fatalf("unexpected external proxy defaults: %+v", settings.Proxy.ExternalProxy)
	}
	if settings.Proxy.WebInterface.Port != 8889 {
		t.Fatalf("unexpected web interface defaults: %+v", settings.Proxy.WebInterface)
	}
	if !settings.Window.StructureView {
		t.Fatalf("expected structure view default true")
	}
}

func TestLoadPacketMindSettings_MissingFileReturnsDefaults(t *testing.T) {
	settings, err := LoadPacketMindSettings(t.TempDir())
	if err != nil {
		t.Fatalf("LoadPacketMindSettings failed: %v", err)
	}
	if settings.Proxy.Listener.Port != 8888 || settings.Cert.Organization != "PacketMind" {
		t.Fatalf("unexpected defaults from missing file: %+v", settings)
	}
}

func TestAppSettingsStore_UpdatePreservesPasswordWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	settings := DefaultPacketMindSettings()
	settings.Proxy.ExternalProxy.Password = "secret"
	store := NewAppSettingsStore(dir, settings)

	next := *DefaultPacketMindSettings()
	next.Proxy.ExternalProxy.Enabled = true
	next.Proxy.ExternalProxy.Password = ""
	updated, err := store.Update(next)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Proxy.ExternalProxy.Password != "secret" {
		t.Fatalf("expected preserved password, got %q", updated.Proxy.ExternalProxy.Password)
	}
}

func TestModelsStore_UpdateSettings_PersistsProviderEntries(t *testing.T) {
	dir := t.TempDir()
	store := NewModelsStore(dir, &ModelsConfig{
		Providers: []ProviderConfig{
			{ID: "mock", Name: "Mock (Demo)", APIType: "mock", Models: map[string]ModelConfig{
				"mock": {Name: "mock"},
			}},
			{ID: "provider-1", Name: "Provider 1", APIType: "openai-compatible", APIKey: "", BaseURL: "https://example.local", Models: map[string]ModelConfig{}},
		},
		DefaultModel:      "mock",
		ActiveProvider:    "mock",
		ActiveModel:       "mock",
		ActiveMaxTokens:   2000,
		ActiveTemperature: 0.7,
	})

	provider := "provider-1"
	apiType := "openai-compatible"
	model := "gpt-test"
	apiKey := "k-123"
	baseURL := "https://example.local"
	maxTokens := 3333
	temp := 0.4

	updated, err := store.UpdateSettings(UpdateAgentSettings{
		Provider:    &provider,
		APIType:     &apiType,
		Model:       &model,
		APIKey:      &apiKey,
		BaseURL:     &baseURL,
		MaxTokens:   &maxTokens,
		Temperature: &temp,
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	if updated.Provider != provider || updated.APIType != apiType || updated.Model != model {
		t.Fatalf("unexpected updated settings: %+v", updated)
	}

	reloaded, err := LoadModels(dir)
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}
	if reloaded.ActiveProvider != provider || reloaded.ActiveModel != model {
		t.Fatalf("expected active provider/model persisted, got %+v", reloaded)
	}
	persisted := reloaded.GetProviderByID(provider)
	if persisted == nil {
		t.Fatalf("expected persisted provider entry")
	}
	if persisted.APIType != apiType || persisted.APIKey != apiKey || persisted.BaseURL != baseURL {
		t.Fatalf("unexpected persisted provider: %+v", persisted)
	}

	if _, err := store.UpdateSettings(UpdateAgentSettings{Provider: ptrString("missing")}); err == nil {
		t.Fatalf("expected error on invalid provider")
	}

	if _, err := LoadModels(dir); err != nil {
		t.Fatalf("expected models.json still valid: %v", err)
	}
}

func TestModelsStore_UpsertProviderRequiresBaseURLForOpenAICompatible(t *testing.T) {
	store := NewModelsStore(t.TempDir(), &ModelsConfig{})
	_, err := store.UpsertProvider(ProviderConfig{ID: "provider-1", APIType: "openai-compatible", APIKey: "k"})
	if err == nil || !strings.Contains(err.Error(), "base_url is required") {
		t.Fatalf("expected base_url validation error, got %v", err)
	}
}

func TestModelsStore_GetProvidersReportsBackendDeletableState(t *testing.T) {
	dir := t.TempDir()
	store := NewModelsStore(dir, &ModelsConfig{
		Providers: []ProviderConfig{
			{ID: "provider-1", Name: "Provider 1", APIType: "openai-compatible", APIKey: "k", BaseURL: "https://example.local", Models: map[string]ModelConfig{
				"base-model": {Name: "base-model"},
			}},
			{ID: "openai", Name: "OpenAI", APIType: "openai-compatible", BaseURL: "https://api.openai.com/v1", Models: map[string]ModelConfig{}},
		},
		DefaultModel:   "base-model",
		ActiveProvider: "provider-1",
		ActiveModel:    "base-model",
	})

	providers := store.GetProviders()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	for _, provider := range providers {
		if !provider.Deletable {
			t.Fatalf("expected all providers to be deletable, got non-deletable: %s", provider.ID)
		}
	}
}

func TestModelsConfig_NormalizeFallsBackToFirstProvider(t *testing.T) {
	cfg := &ModelsConfig{
		Providers: []ProviderConfig{
			{ID: "ark", Name: "Ark", APIType: "openai-compatible", BaseURL: "https://ark.example/v1", Models: map[string]ModelConfig{
				"base-model": {Name: "base-model"},
			}},
		},
		DefaultModel: "base-model",
		ActiveModel:  "base-model",
	}

	cfg.normalize()

	if cfg.ActiveProvider != "ark" {
		t.Fatalf("ActiveProvider = %q, want ark", cfg.ActiveProvider)
	}
	settings := cfg.currentAISettingsLocked()
	if settings.Provider != "ark" {
		t.Fatalf("settings.Provider = %q, want ark", settings.Provider)
	}
}

func TestModelsConfig_NormalizeKeepsProviderEmptyWhenNoProviders(t *testing.T) {
	cfg := &ModelsConfig{
		DefaultModel: "",
		ActiveModel:  "",
	}

	cfg.normalize()

	if cfg.ActiveProvider != "" {
		t.Fatalf("ActiveProvider = %q, want empty", cfg.ActiveProvider)
	}
	settings := cfg.currentAISettingsLocked()
	if settings.Provider != "" {
		t.Fatalf("settings.Provider = %q, want empty", settings.Provider)
	}
}

func TestModelsConfig_NormalizeFallsBackFromStaleProvider(t *testing.T) {
	cfg := &ModelsConfig{
		Providers: []ProviderConfig{
			{ID: "ark", Name: "Ark", APIType: "openai-compatible", BaseURL: "https://ark.example/v1", Models: map[string]ModelConfig{
				"base-model": {Name: "base-model"},
			}},
		},
		DefaultModel:   "base-model",
		ActiveProvider: "zhipu",
		ActiveModel:    "base-model",
	}

	cfg.normalize()

	if cfg.ActiveProvider != "ark" {
		t.Fatalf("ActiveProvider = %q, want ark", cfg.ActiveProvider)
	}
	settings := cfg.currentAISettingsLocked()
	if settings.Provider != "ark" {
		t.Fatalf("settings.Provider = %q, want ark", settings.Provider)
	}
}

func ptrString(s string) *string { return &s }
