package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/packetmind/packetmind/internal/agent"
)

var DefaultModelsStore *ModelsStore

type ModelsStore struct {
	mu     sync.RWMutex
	path   string
	models *ModelsConfig
}

func LoadModels(configPath string) (*ModelsConfig, error) {
	modelsPath := filepath.Join(configPath, "models.json")

	data, err := os.ReadFile(modelsPath)
	if err != nil {
		return nil, err
	}

	var modelsCfg ModelsConfig
	if err := json.Unmarshal(data, &modelsCfg); err != nil {
		return nil, err
	}
	modelsCfg.normalize()

	return &modelsCfg, nil
}

func NewModelsStore(configPath string, modelsCfg *ModelsConfig) *ModelsStore {
	cloned := cloneModelsConfig(modelsCfg)
	cloned.normalize()
	return &ModelsStore{
		path:   filepath.Join(configPath, "models.json"),
		models: cloned,
	}
}

func (s *ModelsStore) Snapshot() *ModelsConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneModelsConfig(s.models)
}

func (s *ModelsStore) GetModelByID(id string) *ModelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models.GetModelByID(id)
}

func (s *ModelsStore) GetModelByProviderAndID(provider, id string) *ModelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models.GetModelByProviderAndID(provider, id)
}

func (s *ModelsStore) GetSettings() AgentSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models.currentAISettingsLocked()
}

func (s *ModelsStore) UpdateSettings(req UpdateAgentSettings) (AgentSettings, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	settings := s.models.currentAISettingsLocked()

	if req.Provider != nil {
		settings.Provider = strings.TrimSpace(*req.Provider)
		providerCfg := s.models.GetProviderByID(settings.Provider)
		if providerCfg == nil {
			return AgentSettings{}, fmt.Errorf("invalid provider id")
		}
		settings.APIType = providerCfg.APIType
	}
	if req.APIType != nil {
		settings.APIType = strings.TrimSpace(*req.APIType)
	}

	if req.Model != nil {
		modelID := strings.TrimSpace(*req.Model)
		if modelID == "" {
			return AgentSettings{}, fmt.Errorf("invalid model id")
		}
		settings.Model = modelID
		if modelCfg := s.models.GetModelByID(modelID); modelCfg != nil && modelCfg.Limit != nil {
			if settings.MaxTokens <= 0 {
				settings.MaxTokens = modelCfg.Limit.Output
			}
		}
	}

	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		settings.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil && *req.Temperature >= 0 {
		settings.Temperature = *req.Temperature
	}
	if req.BaseURL != nil {
		settings.BaseURL = strings.TrimSpace(*req.BaseURL)
	}

	if settings.Model == "" {
		settings.Model = s.models.DefaultModel
	}
	if settings.Provider == "" {
		settings.Provider = s.models.firstProviderID()
	}
	if settings.APIType == "" {
		if providerCfg := s.models.GetProviderByID(settings.Provider); providerCfg != nil {
			settings.APIType = providerCfg.APIType
		}
	}
	if settings.APIType == "" {
		settings.APIType = "openai-compatible"
	}

	if req.APIKey != nil || req.BaseURL != nil || req.APIType != nil {
		providerCfg := s.models.GetProviderByID(settings.Provider)
		if providerCfg == nil {
			return AgentSettings{}, fmt.Errorf("invalid provider id")
		}
		updatedProvider := *providerCfg
		if req.APIType != nil {
			updatedProvider.APIType = strings.TrimSpace(*req.APIType)
		}
		if req.APIKey != nil {
			updatedProvider.APIKey = *req.APIKey
		}
		if req.BaseURL != nil {
			updatedProvider.BaseURL = strings.TrimSpace(*req.BaseURL)
		}
		if err := validateProviderConfig(updatedProvider); err != nil {
			return AgentSettings{}, err
		}
		s.models.setProvider(updatedProvider)
		settings.APIType = updatedProvider.APIType
		settings.BaseURL = updatedProvider.BaseURL
	}

	s.models.ActiveProvider = settings.Provider
	s.models.ActiveModel = settings.Model
	s.models.ActiveMaxTokens = settings.MaxTokens
	s.models.ActiveTemperature = settings.Temperature

	s.models.normalize()
	if err := saveModelsConfig(s.path, s.models); err != nil {
		return AgentSettings{}, err
	}

	return s.models.currentAISettingsLocked(), nil
}

func (s *ModelsStore) APIKey(provider string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models.GetAPIKey(provider)
}

func (s *ModelsStore) BaseURL(provider string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models.GetBaseURL(provider)
}

func (s *ModelsStore) APIType(provider string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models.GetAPIType(provider)
}

func (s *ModelsStore) GetProvider(provider string) *ProviderConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models.GetProviderByID(provider)
}

func (s *ModelsStore) UpsertProvider(provider ProviderConfig) (*ProviderConfig, error) {

	provider.ID = strings.TrimSpace(provider.ID)
	provider.Name = strings.TrimSpace(provider.Name)
	provider.APIType = strings.ToLower(strings.TrimSpace(provider.APIType))
	provider.BaseURL = strings.TrimSpace(provider.BaseURL)
	if provider.Name == "" {
		provider.Name = fallbackProviderName(provider.ID)
	}
	if err := validateProviderConfig(provider); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.models.setProvider(provider)
	s.models.normalize()
	if err := saveModelsConfig(s.path, s.models); err != nil {
		return nil, err
	}
	return s.models.GetProviderByID(provider.ID), nil
}

func (s *ModelsStore) DeleteProvider(providerID string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return fmt.Errorf("provider id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.models.deleteProvider(providerID) {
		return fmt.Errorf("provider not found")
	}
	if strings.EqualFold(s.models.ActiveProvider, providerID) {
		s.models.ActiveProvider = ""
		s.models.ActiveModel = s.models.DefaultModel
	}
	s.models.normalize()
	return saveModelsConfig(s.path, s.models)
}

func (s *ModelsStore) GetProviders() []ProviderInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeProvider := s.models.ActiveProvider

	result := make([]ProviderInfo, 0, len(s.models.Providers))
	for _, provider := range s.models.Providers {
		id := strings.ToLower(strings.TrimSpace(provider.ID))
		info := ProviderInfo{
			ID:         id,
			Name:       provider.Name,
			APIType:    provider.APIType,
			HasAPIKey:  strings.TrimSpace(provider.APIKey) != "",
			HasBaseURL: strings.TrimSpace(provider.BaseURL) != "",
			IsActive:   strings.EqualFold(id, activeProvider),
			Deletable:  true,
			ModelCount: len(provider.Models),
		}
		if desc, ok := agent.GetProviderDescriptor(provider.APIType); ok {
			if info.Name == "" {
				info.Name = desc.Name
			}
			info.Icon = desc.Icon
		}
		if info.Name == "" {
			info.Name = fallbackProviderName(id)
		}
		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsActive != result[j].IsActive {
			return result[i].IsActive
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func (s *ModelsStore) GetModelsGrouped() []ModelsByProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ModelsByProvider, 0, len(s.models.Providers))
	for _, provider := range s.models.Providers {
		models := make([]ModelConfig, 0, len(provider.Models))
		for _, m := range provider.Models {
			models = append(models, m)
		}
		entry := ModelsByProvider{
			Provider:     provider.ID,
			ProviderName: provider.Name,
			Models:       models,
		}
		if desc, ok := agent.GetProviderDescriptor(provider.APIType); ok {
			entry.ProviderIcon = desc.Icon
		}
		result = append(result, entry)
	}
	return result
}

func (s *ModelsStore) RuntimeModels() []ModelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var models []ModelConfig
	for i := range s.models.Providers {
		for _, m := range s.models.Providers[i].Models {
			models = append(models, m)
		}
	}
	return models
}

func (s *ModelsStore) UpsertModel(providerID, modelID string, model ModelConfig) error {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	if providerID == "" {
		return fmt.Errorf("provider id is required")
	}
	if modelID == "" {
		return fmt.Errorf("model id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.models.GetProviderByID(providerID)
	if p == nil {
		return fmt.Errorf("provider not found: %s", providerID)
	}
	for i := range s.models.Providers {
		if strings.EqualFold(s.models.Providers[i].ID, providerID) {
			if s.models.Providers[i].Models == nil {
				s.models.Providers[i].Models = make(map[string]ModelConfig)
			}
			s.models.Providers[i].Models[modelID] = model
			break
		}
	}
	s.models.normalize()
	return saveModelsConfig(s.path, s.models)
}

func (s *ModelsStore) DeleteModel(providerID, modelID string) error {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	if providerID == "" {
		return fmt.Errorf("provider id is required")
	}
	if modelID == "" {
		return fmt.Errorf("model id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.models.Providers {
		if strings.EqualFold(s.models.Providers[i].ID, providerID) {
			if s.models.Providers[i].Models == nil {
				return fmt.Errorf("model not found: %s", modelID)
			}
			if _, ok := s.models.Providers[i].Models[modelID]; !ok {
				return fmt.Errorf("model not found: %s", modelID)
			}
			delete(s.models.Providers[i].Models, modelID)
			if strings.EqualFold(s.models.ActiveModel, modelID) {
				s.models.ActiveModel = s.models.DefaultModel
			}
			s.models.normalize()
			return saveModelsConfig(s.path, s.models)
		}
	}
	return fmt.Errorf("provider not found: %s", providerID)
}

func saveModelsConfig(path string, modelsCfg *ModelsConfig) error {
	if path == "" {
		return fmt.Errorf("models config path is empty")
	}
	if modelsCfg == nil {
		return fmt.Errorf("models config is nil")
	}

	payload := cloneModelsConfig(modelsCfg)

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal models config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create models config dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp models config: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to replace models config: %w", err)
	}

	return nil
}

func cloneModelsConfig(modelsCfg *ModelsConfig) *ModelsConfig {
	if modelsCfg == nil {
		return &ModelsConfig{}
	}
	cloned := &ModelsConfig{
		Providers:         make([]ProviderConfig, len(modelsCfg.Providers)),
		DefaultModel:      modelsCfg.DefaultModel,
		ActiveProvider:    modelsCfg.ActiveProvider,
		ActiveModel:       modelsCfg.ActiveModel,
		ActiveMaxTokens:   modelsCfg.ActiveMaxTokens,
		ActiveTemperature: modelsCfg.ActiveTemperature,
	}
	for i, p := range modelsCfg.Providers {
		clonedP := p
		if p.Models != nil {
			clonedP.Models = make(map[string]ModelConfig, len(p.Models))
			for k, v := range p.Models {
				clonedP.Models[k] = v
			}
		}
		cloned.Providers[i] = clonedP
	}
	return cloned
}

func cloneProviderList(providers []ProviderConfig) []ProviderConfig {
	if len(providers) == 0 {
		return nil
	}
	out := make([]ProviderConfig, len(providers))
	copy(out, providers)
	return out
}

func validateProviderConfig(provider ProviderConfig) error {
	if strings.TrimSpace(provider.ID) == "" {
		return fmt.Errorf("provider id is required")
	}
	if strings.TrimSpace(provider.APIType) == "" {
		return fmt.Errorf("api_type is required")
	}
	if _, ok := agent.GetProviderDescriptor(provider.APIType); !ok {
		return fmt.Errorf("unsupported api_type: %s", provider.APIType)
	}
	if strings.EqualFold(provider.APIType, "openai-compatible") && strings.TrimSpace(provider.BaseURL) == "" {
		return fmt.Errorf("base_url is required for openai-compatible providers")
	}
	return nil
}
