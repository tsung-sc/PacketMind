package config

import (
	"sync"
)

var DefaultSettingsStore *AppSettingsStore

type AppSettingsStore struct {
	mu       sync.RWMutex
	path     string
	settings *AppSettings
}

// NewAppSettingsStore creates a new thread-safe app settings store.
func NewAppSettingsStore(configPath string, settings *AppSettings) *AppSettingsStore {
	cloned := cloneAppSettings(settings)
	cloned.normalize()
	return &AppSettingsStore{
		path:     packetMindSettingsPath(configPath),
		settings: cloned,
	}
}

// Snapshot returns a deep copy of the current app settings.
func (s *AppSettingsStore) Snapshot() *AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAppSettings(s.settings)
}

// Update applies new settings and persists them. Empty passwords are preserved.
func (s *AppSettingsStore) Update(next AppSettings) (*AppSettings, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	updated := cloneAppSettings(&next)
	updated.normalize()
	if updated.Proxy.ExternalProxy.Password == "" && s.settings != nil {
		updated.Proxy.ExternalProxy.Password = s.settings.Proxy.ExternalProxy.Password
	}

	if err := SavePacketMindSettings(s.path, updated); err != nil {
		return nil, err
	}
	s.settings = updated
	return cloneAppSettings(updated), nil
}
