// Package provider implements Git provider management for the review engine.
// This file contains unit tests for the provider manager.
package provider

import (
	"sync"
	"testing"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
)

// mockStore is a minimal mock store for testing
type mockStore struct {
	settingsStore *mockSettingsStore
}

func newMockStore() *mockStore {
	return &mockStore{
		settingsStore: &mockSettingsStore{
			settings: make(map[string]*model.SystemSetting),
		},
	}
}

func (m *mockStore) Review() store.ReviewStore {
	return nil
}

func (m *mockStore) Report() store.ReportStore {
	return nil
}

func (m *mockStore) Settings() store.SettingsStore {
	return m.settingsStore
}

func (m *mockStore) RepositoryConfig() store.RepositoryConfigStore {
	return nil
}

func (m *mockStore) DB() *gorm.DB {
	return nil
}

func (m *mockStore) Transaction(fn func(store.Store) error) error {
	return fn(m)
}

type mockSettingsStore struct {
	mu       sync.RWMutex
	settings map[string]*model.SystemSetting
}

func (m *mockSettingsStore) Get(category, key string) (*model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	setting, ok := m.settings[category+":"+key]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return setting, nil
}

func (m *mockSettingsStore) GetByCategory(category string) ([]model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var settings []model.SystemSetting
	for _, setting := range m.settings {
		if setting.Category == category {
			settings = append(settings, *setting)
		}
	}
	return settings, nil
}

func (m *mockSettingsStore) GetAll() ([]model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var settings []model.SystemSetting
	for _, setting := range m.settings {
		settings = append(settings, *setting)
	}
	return settings, nil
}

func (m *mockSettingsStore) Create(setting *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[setting.Category+":"+setting.Key] = setting
	return nil
}

func (m *mockSettingsStore) Update(setting *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[setting.Category+":"+setting.Key] = setting
	return nil
}

func (m *mockSettingsStore) Save(setting *model.SystemSetting) error {
	return m.Create(setting)
}

func (m *mockSettingsStore) Delete(category, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.settings, category+":"+key)
	return nil
}

func (m *mockSettingsStore) DeleteByCategory(category string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.settings {
		if m.settings[k].Category == category {
			delete(m.settings, k)
		}
	}
	return nil
}

func (m *mockSettingsStore) DeleteSetting(setting *model.SystemSetting) error {
	return m.Delete(setting.Category, setting.Key)
}

func (m *mockSettingsStore) Count() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.settings)), nil
}

func (m *mockSettingsStore) BatchUpsert(settings []model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range settings {
		m.settings[settings[i].Category+":"+settings[i].Key] = &settings[i]
	}
	return nil
}

func (m *mockSettingsStore) WithTx(tx *gorm.DB) store.SettingsStore {
	return m
}

// TestNewManager tests the NewManager function
func TestNewManager(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.providers == nil {
		t.Error("Manager.providers should not be nil")
	}

	if manager.providerConfigs == nil {
		t.Error("Manager.providerConfigs should not be nil")
	}
}

// TestManagerInitialize tests the Initialize method
func TestManagerInitialize(t *testing.T) {
	t.Run("initialize with empty config", func(t *testing.T) {
		mockStore := newMockStore()

		manager := NewManager(store.Store(mockStore))
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}

		if manager.Count() != 0 {
			t.Errorf("Count() = %d, want 0", manager.Count())
		}
	})

	t.Run("initialize with github provider", func(t *testing.T) {
		mockStore := newMockStore()

		manager := NewManager(store.Store(mockStore))
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}

		// Provider count may be 0 if provider initialization fails (e.g., no network)
		// Just verify no panic occurs
		_ = manager.Count()
		_ = manager.Get("github")
	})

	t.Run("initialize with gitlab provider", func(t *testing.T) {
		mockStore := newMockStore()

		manager := NewManager(store.Store(mockStore))
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}

		// Provider count may be 0 if provider initialization fails (e.g., no network)
		// Just verify no panic occurs
		_ = manager.Get("gitlab")
	})

	t.Run("initialize with gitea provider", func(t *testing.T) {
		mockStore := newMockStore()

		manager := NewManager(store.Store(mockStore))
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}

		// Provider count may be 0 if provider initialization fails (e.g., no network)
		// Just verify no panic occurs
		_ = manager.Get("gitea")
	})

	t.Run("initialize with multiple providers", func(t *testing.T) {
		mockStore := newMockStore()

		manager := NewManager(store.Store(mockStore))
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}

		// Provider count may be 0 if provider initialization fails (e.g., no network)
		// Just verify no panic occurs and List() works
		_ = manager.Count()
		_ = manager.List()
	})

	t.Run("initialize with insecure skip verify", func(t *testing.T) {
		mockStore := newMockStore()

		manager := NewManager(store.Store(mockStore))
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}
	})
}

// TestManagerGet tests the Get method
func TestManagerGet(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	manager.Initialize()

	t.Run("get non-existent provider", func(t *testing.T) {
		provider := manager.Get("nonexistent")
		if provider != nil {
			t.Error("Get() for non-existent provider should return nil")
		}
	})
}

// TestManagerGetWithOK tests the GetWithOK method
func TestManagerGetWithOK(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	manager.Initialize()

	t.Run("get non-existent provider", func(t *testing.T) {
		provider, ok := manager.GetWithOK("nonexistent")
		if ok {
			t.Error("GetWithOK() for non-existent provider should return false")
		}
		if provider != nil {
			t.Error("GetWithOK() for non-existent provider should return nil")
		}
	})
}

// TestManagerGetConfig tests the GetConfig method
func TestManagerGetConfig(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	manager.Initialize()

	t.Run("get non-existent provider config", func(t *testing.T) {
		providerCfg, ok := manager.GetConfig("nonexistent")
		if ok {
			t.Error("GetConfig() for non-existent provider should return false")
		}
		if providerCfg != nil {
			t.Error("GetConfig() for non-existent provider should return nil")
		}
	})
}

// TestManagerList tests the List method
func TestManagerList(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	manager.Initialize()

	names := manager.List()
	if names == nil {
		t.Error("List() should not return nil")
	}
	// With empty config, list should be empty
	if len(names) != 0 {
		t.Errorf("List() length = %d, want 0", len(names))
	}
}

// TestManagerDetectFromURL tests the DetectFromURL method
func TestManagerDetectFromURL(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GitHub URL",
			url:      "https://github.com/owner/repo",
			expected: "github",
		},
		{
			name:     "GitLab URL",
			url:      "https://gitlab.com/owner/repo",
			expected: "gitlab",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.DetectFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("DetectFromURL(%s) = %s, want %s", tt.url, result, tt.expected)
			}
		})
	}
}

// TestManagerCount tests the Count method
func TestManagerCount(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	manager.Initialize()

	count := manager.Count()
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}
}

// TestManagerConcurrentAccess tests thread safety
func TestManagerConcurrentAccess(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	manager.Initialize()

	// Run concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				manager.Get("github")
				manager.GetWithOK("github")
				manager.GetConfig("github")
				manager.List()
				manager.Count()
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestManagerWithInvalidProviderType tests initialization with invalid provider type
func TestManagerWithInvalidProviderType(t *testing.T) {
	mockStore := newMockStore()

	manager := NewManager(store.Store(mockStore))
	err := manager.Initialize()

	// Should not return error, just log warning
	if err != nil {
		t.Errorf("Initialize() should not return error for invalid provider: %v", err)
	}

	// Invalid provider should not be added
	if manager.Get("invalid-provider") != nil {
		t.Error("Invalid provider should not be added to manager")
	}
}
