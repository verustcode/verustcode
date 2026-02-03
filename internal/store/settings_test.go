package store

import (
	"testing"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// TestSettingsStore_Create tests creating a setting
func TestSettingsStore_Create(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	setting := &model.SystemSetting{
		Category:  "test",
		Key:       "test_key",
		Value:     "test_value",
		ValueType: "string",
	}

	err := store.Settings().Create(setting)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify the setting was created
	retrieved, err := store.Settings().Get("test", "test_key")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Value != "test_value" {
		t.Errorf("Expected Value 'test_value', got '%s'", retrieved.Value)
	}
}

// TestSettingsStore_Get tests retrieving a setting
func TestSettingsStore_Get(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	setting := &model.SystemSetting{
		Category:  "test",
		Key:       "test_key",
		Value:     "test_value",
		ValueType: "string",
	}
	store.Settings().Create(setting)

	// Test retrieving existing setting
	retrieved, err := store.Settings().Get("test", "test_key")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Value != "test_value" {
		t.Errorf("Expected Value 'test_value', got '%s'", retrieved.Value)
	}

	// Test retrieving non-existent setting
	_, err = store.Settings().Get("test", "non-existent")
	if err == nil {
		t.Error("Get() should return error for non-existent setting")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestSettingsStore_GetByCategory tests retrieving settings by category
func TestSettingsStore_GetByCategory(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create settings in different categories
	setting1 := &model.SystemSetting{
		Category:  "test",
		Key:       "key1",
		Value:     "value1",
		ValueType: "string",
	}
	setting2 := &model.SystemSetting{
		Category:  "test",
		Key:       "key2",
		Value:     "value2",
		ValueType: "string",
	}
	setting3 := &model.SystemSetting{
		Category:  "other",
		Key:       "key3",
		Value:     "value3",
		ValueType: "string",
	}

	store.Settings().Create(setting1)
	store.Settings().Create(setting2)
	store.Settings().Create(setting3)

	// Test retrieving by category
	settings, err := store.Settings().GetByCategory("test")
	if err != nil {
		t.Fatalf("GetByCategory() failed: %v", err)
	}

	if len(settings) != 2 {
		t.Errorf("Expected 2 settings, got %d", len(settings))
	}
}

// TestSettingsStore_Update tests updating a setting
func TestSettingsStore_Update(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	setting := &model.SystemSetting{
		Category:  "test",
		Key:       "test_key",
		Value:     "old_value",
		ValueType: "string",
	}
	store.Settings().Create(setting)

	// Update the setting
	setting.Value = "new_value"
	err := store.Settings().Update(setting)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify the update
	retrieved, err := store.Settings().Get("test", "test_key")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Value != "new_value" {
		t.Errorf("Expected Value 'new_value', got '%s'", retrieved.Value)
	}
}

// TestSettingsStore_Delete tests deleting a setting
func TestSettingsStore_Delete(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	setting := &model.SystemSetting{
		Category:  "test",
		Key:       "test_key",
		Value:     "test_value",
		ValueType: "string",
	}
	store.Settings().Create(setting)

	// Delete the setting
	err := store.Settings().Delete("test", "test_key")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify it's deleted
	_, err = store.Settings().Get("test", "test_key")
	if err == nil {
		t.Error("Get() should return error after delete")
	}
}

// TestSettingsStore_Count tests counting settings
func TestSettingsStore_Count(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create multiple settings
	setting1 := &model.SystemSetting{
		Category:  "test",
		Key:       "key1",
		Value:     "value1",
		ValueType: "string",
	}
	setting2 := &model.SystemSetting{
		Category:  "test",
		Key:       "key2",
		Value:     "value2",
		ValueType: "string",
	}

	store.Settings().Create(setting1)
	store.Settings().Create(setting2)

	count, err := store.Settings().Count()
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}
