// Package database provides database initialization and connection management.
package database

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/pkg/logger"
)

// migrateReviewDSLConfig removes the dsl_config column from reviews table.
// This migration is necessary because we've moved to dynamic loading of YAML configs
// instead of storing snapshots in the database.
//
// The migration is idempotent - it checks if the dsl_config field exists before migrating.
func migrateReviewDSLConfig(db *gorm.DB) error {
	logger.Info("Checking reviews table schema for dsl_config column")

	// Check if the table exists
	var tableExists bool
	err := db.Raw("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='reviews'").Scan(&tableExists).Error
	if err != nil {
		logger.Error("Failed to check if reviews table exists", zap.Error(err))
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !tableExists {
		logger.Info("Table reviews does not exist yet, skipping migration")
		return nil
	}

	// Check if the dsl_config column exists
	var columnInfo []struct {
		Name string
	}
	err = db.Raw("PRAGMA table_info(reviews)").Scan(&columnInfo).Error
	if err != nil {
		logger.Error("Failed to get table schema", zap.Error(err))
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Check if dsl_config field exists
	hasDSLConfig := false
	for _, col := range columnInfo {
		if col.Name == "dsl_config" {
			hasDSLConfig = true
			break
		}
	}

	if !hasDSLConfig {
		logger.Info("Table reviews already has the new schema (no dsl_config), skipping migration")
		return nil
	}

	logger.Info("Detected dsl_config column in reviews table, starting migration")

	// Count existing records
	var recordCount int64
	if err := db.Table("reviews").Count(&recordCount).Error; err != nil {
		logger.Error("Failed to count existing records", zap.Error(err))
		return fmt.Errorf("failed to count records: %w", err)
	}

	logger.Info("Migration will affect records", zap.Int64("count", recordCount))

	// Perform migration in a transaction
	err = db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create new table without dsl_config column
		// Note: This matches the current Review model structure (without DSLConfig field)
		// Use standard indentation (4 spaces) to ensure GORM can parse the DDL correctly
		createNewTableSQL := `CREATE TABLE IF NOT EXISTS reviews_new (
    id TEXT PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    ref TEXT NOT NULL,
    commit_sha TEXT,
    pr_number INTEGER,
    pr_url TEXT,
    repo_url TEXT NOT NULL,
    repo_path TEXT,
    source TEXT NOT NULL DEFAULT 'cli',
    triggered_by TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    current_rule_index INTEGER DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    completed_at DATETIME,
    duration INTEGER,
    branch_created_at DATETIME,
    merged_at DATETIME,
    author TEXT,
    revision_count INTEGER DEFAULT 1,
    commit_count INTEGER DEFAULT 0,
    lines_added INTEGER DEFAULT 0,
    lines_deleted INTEGER DEFAULT 0,
    files_changed INTEGER DEFAULT 0,
    error_message TEXT
)`

		if err := tx.Exec(createNewTableSQL).Error; err != nil {
			logger.Error("Failed to create new table", zap.Error(err))
			return fmt.Errorf("failed to create new table: %w", err)
		}

		logger.Info("Created new table reviews_new")

		// Step 2: Copy data from old table to new table (excluding dsl_config)
		copyDataSQL := `
		INSERT INTO reviews_new (
			id, created_at, updated_at, deleted_at,
			ref, commit_sha, pr_number, pr_url,
			repo_url, repo_path, source, triggered_by,
			status, current_rule_index, retry_count,
			started_at, completed_at, duration,
			branch_created_at, merged_at, author,
			revision_count, commit_count,
			lines_added, lines_deleted, files_changed,
			error_message
		)
		SELECT 
			id, created_at, updated_at, deleted_at,
			ref, commit_sha, pr_number, pr_url,
			repo_url, repo_path, source, triggered_by,
			status, current_rule_index, retry_count,
			started_at, completed_at, duration,
			branch_created_at, merged_at, author,
			revision_count, commit_count,
			lines_added, lines_deleted, files_changed,
			error_message
		FROM reviews`

		if err := tx.Exec(copyDataSQL).Error; err != nil {
			logger.Error("Failed to copy data", zap.Error(err))
			return fmt.Errorf("failed to copy data: %w", err)
		}

		logger.Info("Copied data to new table", zap.Int64("records", recordCount))

		// Step 3: Drop old table
		if err := tx.Exec("DROP TABLE reviews").Error; err != nil {
			logger.Error("Failed to drop old table", zap.Error(err))
			return fmt.Errorf("failed to drop old table: %w", err)
		}

		logger.Info("Dropped old table")

		// Step 4: Rename new table to original name
		if err := tx.Exec("ALTER TABLE reviews_new RENAME TO reviews").Error; err != nil {
			logger.Error("Failed to rename new table", zap.Error(err))
			return fmt.Errorf("failed to rename table: %w", err)
		}

		logger.Info("Renamed new table to reviews")

		// Step 5: Recreate indexes
		// Index on commit_sha
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_reviews_commit_sha ON reviews(commit_sha)").Error; err != nil {
			logger.Warn("Failed to create commit_sha index", zap.Error(err))
		}

		// Index on pr_number
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_reviews_pr_number ON reviews(pr_number)").Error; err != nil {
			logger.Warn("Failed to create pr_number index", zap.Error(err))
		}

		// Index on repo_url
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_reviews_repo_url ON reviews(repo_url)").Error; err != nil {
			logger.Warn("Failed to create repo_url index", zap.Error(err))
		}

		// Index on status
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_reviews_status ON reviews(status)").Error; err != nil {
			logger.Warn("Failed to create status index", zap.Error(err))
		}

		// Unique index on pr_url and commit_sha
		if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_pr_url_commit ON reviews(pr_url, commit_sha)").Error; err != nil {
			logger.Warn("Failed to create pr_url_commit unique index", zap.Error(err))
		}

		// Index on deleted_at for soft delete queries
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_reviews_deleted_at ON reviews(deleted_at)").Error; err != nil {
			logger.Warn("Failed to create deleted_at index", zap.Error(err))
		}

		logger.Info("Recreated indexes")

		return nil
	})

	if err != nil {
		logger.Error("Migration failed, transaction rolled back", zap.Error(err))
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.Info("Successfully migrated reviews table to remove dsl_config column",
		zap.Int64("migrated_records", recordCount))

	return nil
}
