// Package database provides database initialization and connection management.
// It uses GORM with SQLite for embedded database storage, with driver abstraction
// for future extensibility to support other relational databases.
package database

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// DefaultDBPath is the hardcoded database file path
	// This path is fixed to prevent data loss from configuration errors
	DefaultDBPath = "./data/verustcode.db"
)

var (
	db   *gorm.DB
	once sync.Once
)

// Init initializes the database connection and performs auto-migration.
// This function is safe to call multiple times; only the first call will take effect.
// The database path is hardcoded to DefaultDBPath to prevent data loss from configuration errors.
func Init() error {
	return InitWithPath(DefaultDBPath)
}

// InitWithPath initializes the database with a custom path.
// This function is primarily for testing purposes.
// For production use, call Init() instead which uses the hardcoded path.
func InitWithPath(dbPath string) error {
	var initErr error
	once.Do(func() {
		initErr = initDB(dbPath)
	})
	return initErr
}

// initDB creates the database connection and runs migrations
func initDB(dbPath string) error {
	logger.Info("Initializing database", zap.String("path", dbPath))

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("Failed to create database directory", zap.Error(err), zap.String("dir", dir))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to create database directory", err)
	}

	// 创建SQLite驱动（当前只支持SQLite）
	// Create SQLite driver (currently only SQLite is supported)
	driver := &SQLiteDriver{}

	// Configure GORM logger
	gormLog := gormlogger.Default.LogMode(gormlogger.Silent)

	// Open database connection using driver
	dialector, err := driver.Open(dbPath)
	if err != nil {
		logger.Error("Failed to open database", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to open database", err)
	}

	db, err = gorm.Open(dialector, &gorm.Config{
		Logger: gormLog,
	})
	if err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to connect to database", err)
	}

	// 迁移前配置：连接池、WAL模式等（不启用外键约束）
	// Apply pre-migration configurations: connection pool, WAL mode, etc. (foreign keys disabled)
	if err := driver.PreMigrationConfig(db); err != nil {
		logger.Error("Failed to apply pre-migration config", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to apply pre-migration config", err)
	}

	// 执行数据库迁移（此时外键约束未启用，避免孤儿记录导致迁移失败）
	// Run auto-migration (foreign keys disabled to avoid orphan record failures)
	if err := migrate(); err != nil {
		return err
	}

	// 迁移后配置：启用外键约束
	// Apply post-migration configurations: enable foreign key constraints
	if err := driver.PostMigrationConfig(db); err != nil {
		logger.Error("Failed to apply post-migration config", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to apply post-migration config", err)
	}

	logger.Info("Database initialized successfully", zap.String("driver", driver.Name()))
	return nil
}

// migrate runs auto-migration for all models
func migrate() error {
	logger.Info("Running database migrations")

	// Run custom migrations before GORM auto-migration
	// This ensures table schemas are compatible with model definitions
	if err := migrateRepositoryReviewConfigs(db); err != nil {
		logger.Error("Failed to migrate repository_review_configs table", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBMigration, "failed to migrate repository_review_configs table", err)
	}

	if err := migrateReviewDSLConfig(db); err != nil {
		logger.Error("Failed to migrate reviews table", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBMigration, "failed to migrate reviews table", err)
	}

	// Migrate to remove summary columns from review_rules and review_rule_runs
	if err := migrateRemoveSummaryColumns(db); err != nil {
		logger.Error("Failed to migrate to remove summary columns", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBMigration, "failed to migrate to remove summary columns", err)
	}

	models := model.AllModels()
	if err := db.AutoMigrate(models...); err != nil {
		logger.Error("Failed to run database migrations", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBMigration, "failed to run database migrations", err)
	}

	logger.Info("Database migrations completed", zap.Int("models", len(models)))

	// Run data fixes after schema migration
	if err := fixGitProvidersData(); err != nil {
		logger.Warn("Failed to fix git providers data", zap.Error(err))
		// Non-fatal error, continue with startup
	}

	return nil
}

// fixGitProvidersData fixes corrupted git.providers data in the database.
// If providers is stored as an invalid format (not a valid JSON array),
// it will be reset to an empty array.
func fixGitProvidersData() error {
	var setting model.SystemSetting
	result := db.Where("category = ? AND key = ?", "git", "providers").First(&setting)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// No providers setting, nothing to fix
			return nil
		}
		return result.Error
	}

	// Check if the value is a valid JSON array
	var providers []interface{}
	if err := json.Unmarshal([]byte(setting.Value), &providers); err != nil {
		// Value is not a valid JSON array, reset to empty array
		logger.Info("Fixing corrupted git.providers data",
			zap.String("old_value", setting.Value),
			zap.String("new_value", "[]"))

		setting.Value = "[]"
		setting.ValueType = "array"
		if err := db.Save(&setting).Error; err != nil {
			logger.Error("Failed to reset git.providers", zap.Error(err))
			return err
		}

		logger.Info("Git providers data fixed successfully")
	}

	return nil
}

// Get returns the database instance.
// Panics if the database hasn't been initialized.
func Get() *gorm.DB {
	if db == nil {
		panic("database not initialized, call Init first")
	}
	return db
}

// Close closes the database connection
func Close() error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	logger.Info("Closing database connection")
	return sqlDB.Close()
}

// ResetForTesting resets the database state for testing purposes.
// This allows re-initialization of the database in tests.
// WARNING: Only use this function in tests!
func ResetForTesting() {
	if db != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		db = nil
	}
	once = sync.Once{}
}

// Transaction executes a function within a database transaction
func Transaction(fn func(tx *gorm.DB) error) error {
	return Get().Transaction(fn)
}

// HealthCheck performs a simple health check on the database
func HealthCheck() error {
	sqlDB, err := db.DB()
	if err != nil {
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to get database connection", err)
	}
	return sqlDB.Ping()
}
