// Package database provides database initialization and connection management.
package database

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/pkg/logger"
)

// migrateRepositoryReviewConfigs migrates the repository_review_configs table
// from the old schema (with provider, owner, repo fields) to the new schema
// (with only repo_url, review_file, description fields).
//
// This migration is necessary because the old schema had NOT NULL constraints
// on provider, owner, repo fields that no longer exist in the model definition.
//
// The migration is idempotent - it checks if the old fields exist before migrating.
func migrateRepositoryReviewConfigs(db *gorm.DB) error {
	logger.Info("Checking repository_review_configs table schema")

	// Check if the table exists
	var tableExists bool
	err := db.Raw("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='repository_review_configs'").Scan(&tableExists).Error
	if err != nil {
		logger.Error("Failed to check if repository_review_configs table exists", zap.Error(err))
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !tableExists {
		logger.Info("Table repository_review_configs does not exist yet, skipping migration")
		return nil
	}

	// Check if the old schema fields exist
	var columnInfo []struct {
		Name string
	}
	err = db.Raw("PRAGMA table_info(repository_review_configs)").Scan(&columnInfo).Error
	if err != nil {
		logger.Error("Failed to get table schema", zap.Error(err))
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Check if old fields (provider, owner, repo) exist
	hasOldFields := false
	for _, col := range columnInfo {
		if col.Name == "provider" || col.Name == "owner" || col.Name == "repo" {
			hasOldFields = true
			break
		}
	}

	if !hasOldFields {
		logger.Info("Table repository_review_configs already has the new schema, skipping migration")
		return nil
	}

	logger.Info("Detected old schema in repository_review_configs, starting migration")

	// Count existing records
	var recordCount int64
	if err := db.Table("repository_review_configs").Count(&recordCount).Error; err != nil {
		logger.Error("Failed to count existing records", zap.Error(err))
		return fmt.Errorf("failed to count records: %w", err)
	}

	logger.Info("Migration will affect records", zap.Int64("count", recordCount))

	// Perform migration in a transaction
	err = db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create new table with correct schema
		createNewTableSQL := `
		CREATE TABLE IF NOT EXISTS repository_review_configs_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			repo_url TEXT NOT NULL,
			review_file TEXT,
			description TEXT
		)`

		if err := tx.Exec(createNewTableSQL).Error; err != nil {
			logger.Error("Failed to create new table", zap.Error(err))
			return fmt.Errorf("failed to create new table: %w", err)
		}

		logger.Info("Created new table repository_review_configs_new")

		// Step 2: Copy data from old table to new table
		// Only copy the fields that exist in both schemas
		copyDataSQL := `
		INSERT INTO repository_review_configs_new (id, created_at, updated_at, deleted_at, repo_url, review_file, description)
		SELECT id, created_at, updated_at, deleted_at, repo_url, review_file, description
		FROM repository_review_configs`

		if err := tx.Exec(copyDataSQL).Error; err != nil {
			logger.Error("Failed to copy data", zap.Error(err))
			return fmt.Errorf("failed to copy data: %w", err)
		}

		logger.Info("Copied data to new table", zap.Int64("records", recordCount))

		// Step 3: Drop old table
		if err := tx.Exec("DROP TABLE repository_review_configs").Error; err != nil {
			logger.Error("Failed to drop old table", zap.Error(err))
			return fmt.Errorf("failed to drop old table: %w", err)
		}

		logger.Info("Dropped old table")

		// Step 4: Rename new table to original name
		if err := tx.Exec("ALTER TABLE repository_review_configs_new RENAME TO repository_review_configs").Error; err != nil {
			logger.Error("Failed to rename new table", zap.Error(err))
			return fmt.Errorf("failed to rename table: %w", err)
		}

		logger.Info("Renamed new table to repository_review_configs")

		// Step 5: Recreate indexes
		// Create unique index on repo_url
		createIndexSQL := `CREATE UNIQUE INDEX idx_repository_review_configs_repo_url ON repository_review_configs(repo_url)`
		if err := tx.Exec(createIndexSQL).Error; err != nil {
			logger.Error("Failed to create unique index", zap.Error(err))
			return fmt.Errorf("failed to create unique index: %w", err)
		}

		// Create index on deleted_at for soft delete queries
		createDeletedAtIndexSQL := `CREATE INDEX idx_repository_review_configs_deleted_at ON repository_review_configs(deleted_at)`
		if err := tx.Exec(createDeletedAtIndexSQL).Error; err != nil {
			logger.Error("Failed to create deleted_at index", zap.Error(err))
			return fmt.Errorf("failed to create deleted_at index: %w", err)
		}

		logger.Info("Recreated indexes")

		return nil
	})

	if err != nil {
		logger.Error("Migration failed, transaction rolled back", zap.Error(err))
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.Info("Successfully migrated repository_review_configs table to new schema",
		zap.Int64("migrated_records", recordCount))

	return nil
}
