package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/pkg/logger"
)

func TestSQLiteOptimizations(t *testing.T) {
	// Initialize logger for testing
	logger.Init(logger.Config{
		Level:  "info",
		Format: "text",
		File:   "",
	})
	defer logger.Sync()

	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize database with custom path for testing
	err := InitWithPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		Close()
		os.Remove(dbPath)
	}()

	// Get database connection
	db := Get()

	// Check journal_mode (should be WAL)
	var journalMode string
	result := db.Raw("PRAGMA journal_mode").Scan(&journalMode)
	if result.Error != nil {
		t.Fatalf("Failed to query journal_mode: %v", result.Error)
	}
	if journalMode != "wal" {
		t.Errorf("Expected journal_mode to be 'wal', got '%s'", journalMode)
	}

	// Check synchronous (should be 1 for NORMAL)
	var synchronous int
	result = db.Raw("PRAGMA synchronous").Scan(&synchronous)
	if result.Error != nil {
		t.Fatalf("Failed to query synchronous: %v", result.Error)
	}
	if synchronous != 1 {
		t.Errorf("Expected synchronous to be 1 (NORMAL), got %d", synchronous)
	}

	// Check foreign_keys (should be ON)
	var foreignKeys int
	result = db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys)
	if result.Error != nil {
		t.Fatalf("Failed to query foreign_keys: %v", result.Error)
	}
	if foreignKeys != 1 {
		t.Errorf("Expected foreign_keys to be 1 (ON), got %d", foreignKeys)
	}

	t.Logf("SQLite optimizations verified: journal_mode=%s, synchronous=%d, foreign_keys=%d",
		journalMode, synchronous, foreignKeys)
}

// TestMigrateRepositoryReviewConfigs_TableNotExists tests migration when table doesn't exist
func TestMigrateRepositoryReviewConfigs_TableNotExists(t *testing.T) {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
	defer logger.Sync()

	// Reset database state for testing
	ResetForTesting()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := InitWithPath(dbPath)
	require.NoError(t, err)
	defer Close()

	db := Get()

	// Drop table if it exists (to simulate table not existing)
	db.Exec("DROP TABLE IF EXISTS repository_review_configs")

	// Run migration - should succeed without error (migration skips when table doesn't exist)
	err = migrateRepositoryReviewConfigs(db)
	assert.NoError(t, err)

	// Table should not exist after migration (migration doesn't create tables)
	var tableExists bool
	err = db.Raw("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='repository_review_configs'").Scan(&tableExists).Error
	assert.NoError(t, err)
	assert.False(t, tableExists, "Table should not exist after migration when it didn't exist before")
}

// TestMigrateRepositoryReviewConfigs_NewSchema tests migration when table already has new schema
func TestMigrateRepositoryReviewConfigs_NewSchema(t *testing.T) {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
	defer logger.Sync()

	// Reset database state for testing
	ResetForTesting()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := InitWithPath(dbPath)
	require.NoError(t, err)
	defer Close()

	db := Get()

	// Ensure table exists with new schema (GORM will create it)
	config := &model.RepositoryReviewConfig{
		RepoURL: "https://github.com/test/repo",
	}
	err = db.Create(config).Error
	require.NoError(t, err)

	// Run migration - should skip (already new schema)
	err = migrateRepositoryReviewConfigs(db)
	assert.NoError(t, err)

	// Verify table still exists and has correct schema
	var count int64
	err = db.Model(&model.RepositoryReviewConfig{}).Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// TestMigrateRepositoryReviewConfigs_OldSchema tests migration from old schema to new schema
func TestMigrateRepositoryReviewConfigs_OldSchema(t *testing.T) {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
	defer logger.Sync()

	// Reset database state for testing
	ResetForTesting()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := InitWithPath(dbPath)
	require.NoError(t, err)
	defer Close()

	db := Get()

	// Drop existing table
	db.Exec("DROP TABLE IF EXISTS repository_review_configs")

	// Create old schema table manually
	createOldTableSQL := `
	CREATE TABLE repository_review_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		provider TEXT NOT NULL,
		owner TEXT NOT NULL,
		repo TEXT NOT NULL,
		repo_url TEXT,
		review_file TEXT,
		description TEXT
	)`
	err = db.Exec(createOldTableSQL).Error
	require.NoError(t, err)

	// Insert test data
	insertSQL := `
	INSERT INTO repository_review_configs (provider, owner, repo, repo_url, review_file, description)
	VALUES ('github', 'test-owner', 'test-repo', 'https://github.com/test-owner/test-repo', 'review.yaml', 'Test description')`
	err = db.Exec(insertSQL).Error
	require.NoError(t, err)

	// Run migration
	err = migrateRepositoryReviewConfigs(db)
	require.NoError(t, err)

	// Verify migration succeeded
	var config model.RepositoryReviewConfig
	err = db.Where("repo_url = ?", "https://github.com/test-owner/test-repo").First(&config).Error
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/test-owner/test-repo", config.RepoURL)
	assert.Equal(t, "review.yaml", config.ReviewFile)
	assert.Equal(t, "Test description", config.Description)

	// Verify old fields don't exist
	var columnInfo []struct {
		Name string
	}
	err = db.Raw("PRAGMA table_info(repository_review_configs)").Scan(&columnInfo).Error
	require.NoError(t, err)

	hasOldFields := false
	for _, col := range columnInfo {
		if col.Name == "provider" || col.Name == "owner" || col.Name == "repo" {
			hasOldFields = true
			break
		}
	}
	assert.False(t, hasOldFields, "Old fields should not exist after migration")

	// Verify indexes exist
	var indexInfo []struct {
		Name string
	}
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='repository_review_configs'").Scan(&indexInfo).Error
	require.NoError(t, err)

	hasRepoURLIndex := false
	hasDeletedAtIndex := false
	for _, idx := range indexInfo {
		if idx.Name == "idx_repository_review_configs_repo_url" {
			hasRepoURLIndex = true
		}
		if idx.Name == "idx_repository_review_configs_deleted_at" {
			hasDeletedAtIndex = true
		}
	}
	assert.True(t, hasRepoURLIndex, "repo_url index should exist")
	assert.True(t, hasDeletedAtIndex, "deleted_at index should exist")
}

// TestMigrateRepositoryReviewConfigs_MultipleRecords tests migration with multiple records
func TestMigrateRepositoryReviewConfigs_MultipleRecords(t *testing.T) {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
	defer logger.Sync()

	// Reset database state for testing
	ResetForTesting()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := InitWithPath(dbPath)
	require.NoError(t, err)
	defer Close()

	db := Get()

	// Drop existing table
	db.Exec("DROP TABLE IF EXISTS repository_review_configs")

	// Create old schema table
	createOldTableSQL := `
	CREATE TABLE repository_review_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		provider TEXT NOT NULL,
		owner TEXT NOT NULL,
		repo TEXT NOT NULL,
		repo_url TEXT,
		review_file TEXT,
		description TEXT
	)`
	err = db.Exec(createOldTableSQL).Error
	require.NoError(t, err)

	// Insert multiple test records
	insertSQL1 := `
	INSERT INTO repository_review_configs (provider, owner, repo, repo_url, review_file, description)
	VALUES ('github', 'owner1', 'repo1', 'https://github.com/owner1/repo1', 'review1.yaml', 'Desc 1')`
	err = db.Exec(insertSQL1).Error
	require.NoError(t, err)

	insertSQL2 := `
	INSERT INTO repository_review_configs (provider, owner, repo, repo_url, review_file, description)
	VALUES ('gitlab', 'owner2', 'repo2', 'https://gitlab.com/owner2/repo2', 'review2.yaml', 'Desc 2')`
	err = db.Exec(insertSQL2).Error
	require.NoError(t, err)

	// Run migration
	err = migrateRepositoryReviewConfigs(db)
	require.NoError(t, err)

	// Verify all records migrated
	var count int64
	err = db.Model(&model.RepositoryReviewConfig{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify records are accessible
	var config1, config2 model.RepositoryReviewConfig
	err = db.Where("repo_url = ?", "https://github.com/owner1/repo1").First(&config1).Error
	require.NoError(t, err)
	assert.Equal(t, "review1.yaml", config1.ReviewFile)

	err = db.Where("repo_url = ?", "https://gitlab.com/owner2/repo2").First(&config2).Error
	require.NoError(t, err)
	assert.Equal(t, "review2.yaml", config2.ReviewFile)
}

// TestMigrateRepositoryReviewConfigs_EmptyTable tests migration with empty table
func TestMigrateRepositoryReviewConfigs_EmptyTable(t *testing.T) {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
	defer logger.Sync()

	// Reset database state for testing
	ResetForTesting()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := InitWithPath(dbPath)
	require.NoError(t, err)
	defer Close()

	db := Get()

	// Drop existing table
	db.Exec("DROP TABLE IF EXISTS repository_review_configs")

	// Create old schema table (empty)
	createOldTableSQL := `
	CREATE TABLE repository_review_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		provider TEXT NOT NULL,
		owner TEXT NOT NULL,
		repo TEXT NOT NULL,
		repo_url TEXT,
		review_file TEXT,
		description TEXT
	)`
	err = db.Exec(createOldTableSQL).Error
	require.NoError(t, err)

	// Run migration
	err = migrateRepositoryReviewConfigs(db)
	require.NoError(t, err)

	// Verify table exists with new schema
	var count int64
	err = db.Model(&model.RepositoryReviewConfig{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// TestMigrateRepositoryReviewConfigs_Idempotent tests that migration is idempotent
func TestMigrateRepositoryReviewConfigs_Idempotent(t *testing.T) {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
	defer logger.Sync()

	// Reset database state for testing
	ResetForTesting()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := InitWithPath(dbPath)
	require.NoError(t, err)
	defer Close()

	db := Get()

	// Ensure table exists with new schema
	config := &model.RepositoryReviewConfig{
		RepoURL: "https://github.com/test/repo",
	}
	err = db.Create(config).Error
	require.NoError(t, err)

	// Run migration multiple times - should all succeed
	err = migrateRepositoryReviewConfigs(db)
	assert.NoError(t, err)

	err = migrateRepositoryReviewConfigs(db)
	assert.NoError(t, err)

	err = migrateRepositoryReviewConfigs(db)
	assert.NoError(t, err)

	// Verify data is still intact
	var count int64
	err = db.Model(&model.RepositoryReviewConfig{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// TestMigrateRepositoryReviewConfigs_WithSoftDelete tests migration with soft-deleted records
func TestMigrateRepositoryReviewConfigs_WithSoftDelete(t *testing.T) {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
	defer logger.Sync()

	// Reset database state for testing
	ResetForTesting()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := InitWithPath(dbPath)
	require.NoError(t, err)
	defer Close()

	db := Get()

	// Drop existing table
	db.Exec("DROP TABLE IF EXISTS repository_review_configs")

	// Create old schema table
	createOldTableSQL := `
	CREATE TABLE repository_review_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		provider TEXT NOT NULL,
		owner TEXT NOT NULL,
		repo TEXT NOT NULL,
		repo_url TEXT,
		review_file TEXT,
		description TEXT
	)`
	err = db.Exec(createOldTableSQL).Error
	require.NoError(t, err)

	// Insert records including one with deleted_at set
	insertSQL1 := `
	INSERT INTO repository_review_configs (provider, owner, repo, repo_url, review_file, deleted_at)
	VALUES ('github', 'owner1', 'repo1', 'https://github.com/owner1/repo1', 'review1.yaml', NULL)`
	err = db.Exec(insertSQL1).Error
	require.NoError(t, err)

	insertSQL2 := `
	INSERT INTO repository_review_configs (provider, owner, repo, repo_url, review_file, deleted_at)
	VALUES ('github', 'owner2', 'repo2', 'https://github.com/owner2/repo2', 'review2.yaml', datetime('now'))`
	err = db.Exec(insertSQL2).Error
	require.NoError(t, err)

	// Run migration
	err = migrateRepositoryReviewConfigs(db)
	require.NoError(t, err)

	// Verify all records migrated (including soft-deleted)
	var count int64
	err = db.Unscoped().Model(&model.RepositoryReviewConfig{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify soft-deleted record is still soft-deleted
	var deletedConfig model.RepositoryReviewConfig
	err = db.Unscoped().Where("repo_url = ?", "https://github.com/owner2/repo2").First(&deletedConfig).Error
	require.NoError(t, err)
	assert.NotNil(t, deletedConfig.DeletedAt)
}
