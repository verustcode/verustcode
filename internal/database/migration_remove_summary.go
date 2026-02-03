// Package database provides database initialization and connection management.
package database

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/pkg/logger"
)

// migrateRemoveSummaryColumns removes the summary column from review_rules and review_rule_runs tables.
// This migration is necessary because summary data is now stored in ReviewResult.Data.
//
// The migration is idempotent - it checks if the summary field exists before migrating.
func migrateRemoveSummaryColumns(db *gorm.DB) error {
	// Migrate review_rules table
	if err := removeSummaryFromTable(db, "review_rules"); err != nil {
		return err
	}

	// Migrate review_rule_runs table
	if err := removeSummaryFromTable(db, "review_rule_runs"); err != nil {
		return err
	}

	return nil
}

// removeSummaryFromTable removes the summary column from the specified table.
func removeSummaryFromTable(db *gorm.DB, tableName string) error {
	logger.Info("Checking table schema for summary column", zap.String("table", tableName))

	// Check if the table exists
	var tableExists bool
	err := db.Raw("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&tableExists).Error
	if err != nil {
		logger.Error("Failed to check if table exists", zap.Error(err), zap.String("table", tableName))
		return fmt.Errorf("failed to check table existence for %s: %w", tableName, err)
	}

	if !tableExists {
		logger.Info("Table does not exist yet, skipping migration", zap.String("table", tableName))
		return nil
	}

	// Check if the summary column exists
	var columnInfo []struct {
		Name string
	}
	err = db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).Scan(&columnInfo).Error
	if err != nil {
		logger.Error("Failed to get table schema", zap.Error(err), zap.String("table", tableName))
		return fmt.Errorf("failed to get table schema for %s: %w", tableName, err)
	}

	// Check if summary field exists
	hasSummary := false
	for _, col := range columnInfo {
		if col.Name == "summary" {
			hasSummary = true
			break
		}
	}

	if !hasSummary {
		logger.Info("Table already has the new schema (no summary), checking structure", zap.String("table", tableName))
		// Even if summary column doesn't exist, check if table structure is correct
		// This handles cases where the table was created with CREATE TABLE ... AS SELECT
		// which doesn't preserve primary keys and constraints
		needsFix, err := checkTableStructure(db, tableName)
		if err != nil {
			logger.Warn("Failed to check table structure", zap.Error(err), zap.String("table", tableName))
			// Continue anyway, let GORM AutoMigrate handle it
		} else if needsFix {
			logger.Warn("Table structure is incomplete (missing primary key). "+
				"Dropping table to let GORM AutoMigrate recreate it with correct structure.",
				zap.String("table", tableName))

			// Drop the table so GORM AutoMigrate can recreate it with correct structure
			if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)).Error; err != nil {
				logger.Error("Failed to drop incomplete table", zap.Error(err), zap.String("table", tableName))
				return fmt.Errorf("failed to drop incomplete table %s: %w", tableName, err)
			}

			logger.Info("Dropped incomplete table, GORM AutoMigrate will recreate it", zap.String("table", tableName))
		}
		return nil
	}

	logger.Info("Detected summary column in table, starting migration", zap.String("table", tableName))

	// Count existing records
	var recordCount int64
	if err := db.Table(tableName).Count(&recordCount).Error; err != nil {
		logger.Error("Failed to count existing records", zap.Error(err), zap.String("table", tableName))
		return fmt.Errorf("failed to count records in %s: %w", tableName, err)
	}

	logger.Info("Migration will affect records", zap.String("table", tableName), zap.Int64("count", recordCount))

	// Get all column names except summary
	var columns []string
	for _, col := range columnInfo {
		if col.Name != "summary" {
			columns = append(columns, col.Name)
		}
	}

	// Build column list for SQL
	columnList := ""
	for i, col := range columns {
		if i > 0 {
			columnList += ", "
		}
		columnList += col
	}

	// Perform migration in a transaction
	err = db.Transaction(func(tx *gorm.DB) error {
		newTableName := tableName + "_new"

		// Step 1: Create new table by copying structure without summary
		// We need to get the original CREATE TABLE statement and remove the summary column
		var createSQL string
		err := tx.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createSQL).Error
		if err != nil {
			logger.Error("Failed to get original CREATE TABLE", zap.Error(err))
			return fmt.Errorf("failed to get CREATE TABLE: %w", err)
		}

		// For simplicity, we'll create the new table by selecting data
		// First, create the new table structure using GORM's auto-migrate (without summary field)
		// This approach uses SELECT INTO which preserves the schema minus the summary column
		createNewTableSQL := fmt.Sprintf("CREATE TABLE %s AS SELECT %s FROM %s WHERE 0", newTableName, columnList, tableName)
		if err := tx.Exec(createNewTableSQL).Error; err != nil {
			logger.Error("Failed to create new table structure", zap.Error(err))
			return fmt.Errorf("failed to create new table structure: %w", err)
		}

		logger.Info("Created new table structure", zap.String("table", newTableName))

		// Step 2: Copy data
		copyDataSQL := fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s", newTableName, columnList, tableName)
		if err := tx.Exec(copyDataSQL).Error; err != nil {
			logger.Error("Failed to copy data", zap.Error(err))
			return fmt.Errorf("failed to copy data: %w", err)
		}

		logger.Info("Copied data to new table", zap.String("table", newTableName), zap.Int64("records", recordCount))

		// Step 3: Drop old table
		if err := tx.Exec(fmt.Sprintf("DROP TABLE %s", tableName)).Error; err != nil {
			logger.Error("Failed to drop old table", zap.Error(err))
			return fmt.Errorf("failed to drop old table: %w", err)
		}

		logger.Info("Dropped old table", zap.String("table", tableName))

		// Step 4: Rename new table
		if err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", newTableName, tableName)).Error; err != nil {
			logger.Error("Failed to rename new table", zap.Error(err))
			return fmt.Errorf("failed to rename table: %w", err)
		}

		logger.Info("Renamed new table", zap.String("table", tableName))

		// Step 5: Recreate indexes based on table
		if tableName == "review_rules" {
			if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_review_rules_review_id ON review_rules(review_id)").Error; err != nil {
				logger.Warn("Failed to create review_id index", zap.Error(err))
			}
			if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_review_rules_rule_id ON review_rules(rule_id)").Error; err != nil {
				logger.Warn("Failed to create rule_id index", zap.Error(err))
			}
			if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_review_rules_status ON review_rules(status)").Error; err != nil {
				logger.Warn("Failed to create status index", zap.Error(err))
			}
			if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_review_rules_deleted_at ON review_rules(deleted_at)").Error; err != nil {
				logger.Warn("Failed to create deleted_at index", zap.Error(err))
			}
		} else if tableName == "review_rule_runs" {
			if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_review_rule_runs_review_rule_id ON review_rule_runs(review_rule_id)").Error; err != nil {
				logger.Warn("Failed to create review_rule_id index", zap.Error(err))
			}
			if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_review_rule_runs_status ON review_rule_runs(status)").Error; err != nil {
				logger.Warn("Failed to create status index", zap.Error(err))
			}
			if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_review_rule_runs_deleted_at ON review_rule_runs(deleted_at)").Error; err != nil {
				logger.Warn("Failed to create deleted_at index", zap.Error(err))
			}
		}

		logger.Info("Recreated indexes", zap.String("table", tableName))

		return nil
	})

	if err != nil {
		logger.Error("Migration failed, transaction rolled back", zap.Error(err), zap.String("table", tableName))
		return fmt.Errorf("migration failed for %s: %w", tableName, err)
	}

	logger.Info("Successfully migrated table to remove summary column",
		zap.String("table", tableName),
		zap.Int64("migrated_records", recordCount))

	// Step 6: Check and fix table structure if needed
	// CREATE TABLE ... AS SELECT doesn't preserve primary keys and constraints,
	// so we need to check if the table structure is correct
	needsFix, err := checkTableStructure(db, tableName)
	if err != nil {
		logger.Warn("Failed to check table structure", zap.Error(err), zap.String("table", tableName))
		// Continue anyway, GORM AutoMigrate will handle it
	} else if needsFix {
		logger.Warn("Table structure is incomplete after migration (missing primary key). "+
			"Dropping table to let GORM AutoMigrate recreate it with correct structure.",
			zap.String("table", tableName))

		// Drop the table so GORM AutoMigrate can recreate it with correct structure
		// This will lose data, but it's necessary to fix the schema
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)).Error; err != nil {
			logger.Error("Failed to drop incomplete table", zap.Error(err), zap.String("table", tableName))
			return fmt.Errorf("failed to drop incomplete table %s: %w", tableName, err)
		}

		logger.Info("Dropped incomplete table, GORM AutoMigrate will recreate it", zap.String("table", tableName))
	}

	return nil
}

// checkTableStructure checks if the table has correct structure (primary key, etc.)
// Returns true if the table structure needs fixing
func checkTableStructure(db *gorm.DB, tableName string) (bool, error) {
	// Check if table has primary key
	var tableInfo []struct {
		Name string
		Type string
		PK   int
	}
	err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).Scan(&tableInfo).Error
	if err != nil {
		return false, fmt.Errorf("failed to check table structure: %w", err)
	}

	hasPK := false
	for _, col := range tableInfo {
		if col.PK > 0 {
			hasPK = true
			break
		}
	}

	// Table structure is incomplete if it doesn't have a primary key
	// This happens when CREATE TABLE ... AS SELECT is used
	return !hasPK, nil
}
