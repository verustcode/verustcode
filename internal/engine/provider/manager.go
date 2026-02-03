// Package provider implements Git provider management for the review engine.
// It handles initialization and access of Git providers.
package provider

import (
	"sync"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine/utils"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Manager handles Git provider initialization and access.
// It is thread-safe and supports concurrent access to providers.
// Manager now uses ConfigProvider for real-time database access to Git provider configurations.
type Manager struct {
	providers       map[string]provider.Provider
	providerConfigs map[string]*config.ProviderConfig
	mu              sync.RWMutex
	configProvider  config.ConfigProvider
}

// NewManager creates a new Provider Manager.
// It now requires a store.Store for real-time database access to Git provider configurations.
func NewManager(s store.Store) *Manager {
	return &Manager{
		providers:       make(map[string]provider.Provider),
		providerConfigs: make(map[string]*config.ProviderConfig),
		configProvider:  config.NewSettingsService(s),
	}
}

// Initialize initializes all Git providers from configuration.
// Should be called once during engine startup.
// Now reads provider configurations directly from database via ConfigProvider.
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get Git providers from database
	providers, err := m.configProvider.GetGitProviders()
	if err != nil {
		logger.Error("Failed to get Git providers from database",
			zap.Error(err),
		)
		return err
	}

	for _, pc := range providers {
		if err := m.initProvider(pc); err != nil {
			logger.Warn("Failed to create provider",
				zap.String("type", pc.Type),
				zap.Error(err),
			)
			continue
		}
	}

	if len(m.providers) == 0 {
		logger.Warn("No Git providers configured")
	}

	return nil
}

// initProvider initializes a single provider (must be called with lock held).
func (m *Manager) initProvider(pc config.ProviderConfig) error {
	// Create a copy of the config to store
	pcCopy := pc

	// Build provider options from config
	opts := &provider.ProviderOptions{
		Token:              pc.Token,
		BaseURL:            pc.URL,
		InsecureSkipVerify: pc.InsecureSkipVerify,
	}

	p, err := provider.Create(pc.Type, opts)
	if err != nil {
		return err
	}

	m.providers[pc.Type] = p
	m.providerConfigs[pc.Type] = &pcCopy
	logger.Info("Initialized Git provider",
		zap.String("type", pc.Type),
		zap.String("url", pc.URL),
		zap.Bool("insecure_skip_verify", pc.InsecureSkipVerify),
	)

	return nil
}

// Get returns a provider by name.
// Thread-safe: uses read lock to protect concurrent access.
// Returns nil if the provider is not found.
func (m *Manager) Get(name string) provider.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providers[name]
}

// GetWithOK returns a provider by name with a boolean indicating if it exists.
// Thread-safe: uses read lock to protect concurrent access.
func (m *Manager) GetWithOK(name string) (provider.Provider, bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.providers[name]

	return p, ok
}

// GetConfig returns the provider configuration by name.
// Thread-safe: uses read lock to protect concurrent access.
func (m *Manager) GetConfig(name string) (*config.ProviderConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.providerConfigs[name]
	return cfg, ok
}

// List returns all available provider names.
// Thread-safe: uses read lock to protect concurrent access.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// DetectFromURL detects the provider name from a repository URL.
// Returns empty string if the provider cannot be detected.
func (m *Manager) DetectFromURL(url string) string {
	return utils.DetectProviderFromURL(url)
}

// Count returns the number of configured providers.
// Thread-safe: uses read lock to protect concurrent access.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.providers)
}

// Refresh reloads all Git providers from the database.
// This can be called when provider configuration changes at runtime.
// Thread-safe: uses write lock to protect concurrent access.
func (m *Manager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get Git providers from database
	providers, err := m.configProvider.GetGitProviders()
	if err != nil {
		logger.Error("Failed to refresh Git providers from database",
			zap.Error(err),
		)
		return err
	}

	// Clear existing providers
	m.providers = make(map[string]provider.Provider)
	m.providerConfigs = make(map[string]*config.ProviderConfig)

	// Reinitialize all providers
	for _, pc := range providers {
		if err := m.initProvider(pc); err != nil {
			logger.Warn("Failed to create provider during refresh",
				zap.String("type", pc.Type),
				zap.Error(err),
			)
			continue
		}
	}

	logger.Info("Git providers refreshed from database",
		zap.Int("count", len(m.providers)),
	)

	return nil
}

// GetConfigFromDB returns the latest provider configuration from database by name.
// This bypasses the cached config and reads directly from the database.
// Thread-safe.
func (m *Manager) GetConfigFromDB(name string) (*config.ProviderConfig, error) {
	providers, err := m.configProvider.GetGitProviders()
	if err != nil {
		return nil, err
	}

	for _, pc := range providers {
		if pc.Type == name {
			return &pc, nil
		}
	}

	return nil, nil
}
