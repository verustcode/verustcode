package store

import (
	"testing"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// TestRepositoryConfigStore_Create tests creating a repository config
func TestRepositoryConfigStore_Create(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	config := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: "default.yaml",
	}

	err := store.RepositoryConfig().Create(config)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify the config was created
	retrieved, err := store.RepositoryConfig().GetByRepoURL("https://github.com/test/repo")
	if err != nil {
		t.Fatalf("GetByRepoURL() failed: %v", err)
	}

	if retrieved.RepoURL != "https://github.com/test/repo" {
		t.Errorf("Expected RepoURL 'https://github.com/test/repo', got '%s'", retrieved.RepoURL)
	}
	if retrieved.ReviewFile != "default.yaml" {
		t.Errorf("Expected ReviewFile 'default.yaml', got '%s'", retrieved.ReviewFile)
	}
}

// TestRepositoryConfigStore_GetByRepoURL tests retrieving a config by repo URL
func TestRepositoryConfigStore_GetByRepoURL(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	config := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: "default.yaml",
	}
	store.RepositoryConfig().Create(config)

	// Test retrieving existing config
	retrieved, err := store.RepositoryConfig().GetByRepoURL("https://github.com/test/repo")
	if err != nil {
		t.Fatalf("GetByRepoURL() failed: %v", err)
	}

	if retrieved.RepoURL != "https://github.com/test/repo" {
		t.Errorf("Expected RepoURL 'https://github.com/test/repo', got '%s'", retrieved.RepoURL)
	}

	// Test retrieving non-existent config
	_, err = store.RepositoryConfig().GetByRepoURL("https://github.com/non-existent/repo")
	if err == nil {
		t.Error("GetByRepoURL() should return error for non-existent config")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestRepositoryConfigStore_Update tests updating a repository config
func TestRepositoryConfigStore_Update(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	config := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: "default.yaml",
	}
	store.RepositoryConfig().Create(config)

	// Update the config
	config.ReviewFile = "custom.yaml"
	config.Description = "Updated description"
	err := store.RepositoryConfig().Update(config)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify the update
	retrieved, err := store.RepositoryConfig().GetByRepoURL("https://github.com/test/repo")
	if err != nil {
		t.Fatalf("GetByRepoURL() failed: %v", err)
	}

	if retrieved.ReviewFile != "custom.yaml" {
		t.Errorf("Expected ReviewFile 'custom.yaml', got '%s'", retrieved.ReviewFile)
	}
	if retrieved.Description != "Updated description" {
		t.Errorf("Expected Description 'Updated description', got '%s'", retrieved.Description)
	}
}

// TestRepositoryConfigStore_Delete tests deleting a repository config
func TestRepositoryConfigStore_Delete(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	config := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: "default.yaml",
	}
	store.RepositoryConfig().Create(config)

	// Delete the config
	err := store.RepositoryConfig().Delete(config.ID)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify it's deleted
	_, err = store.RepositoryConfig().GetByRepoURL("https://github.com/test/repo")
	if err == nil {
		t.Error("GetByRepoURL() should return error after delete")
	}
}

// TestRepositoryConfigStore_DeleteByRepoURL tests deleting by repo URL
func TestRepositoryConfigStore_DeleteByRepoURL(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	config := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: "default.yaml",
	}
	store.RepositoryConfig().Create(config)

	// Delete by repo URL
	err := store.RepositoryConfig().DeleteByRepoURL("https://github.com/test/repo")
	if err != nil {
		t.Fatalf("DeleteByRepoURL() failed: %v", err)
	}

	// Verify it's deleted
	_, err = store.RepositoryConfig().GetByRepoURL("https://github.com/test/repo")
	if err == nil {
		t.Error("GetByRepoURL() should return error after delete")
	}
}

// TestRepositoryConfigStore_ListAll tests listing all repository configs
func TestRepositoryConfigStore_ListAll(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create multiple configs
	config1 := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo1",
		ReviewFile: "default.yaml",
	}
	config2 := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo2",
		ReviewFile: "custom.yaml",
	}

	store.RepositoryConfig().Create(config1)
	store.RepositoryConfig().Create(config2)

	// List all configs
	configs, err := store.RepositoryConfig().ListAll()
	if err != nil {
		t.Fatalf("ListAll() failed: %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("Expected 2 configs, got %d", len(configs))
	}
}
