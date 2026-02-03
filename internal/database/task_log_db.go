// Package database provides database connection and management functionality.
package database

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// DefaultTaskLogDBPath is the hardcoded task log database file path
	// This path is fixed to prevent data loss from configuration errors
	DefaultTaskLogDBPath = "./data/task_logs.db"
)

var (
	taskLogDB   *gorm.DB
	taskLogOnce sync.Once
)

// InitTaskLogDB initializes the task log database connection and performs auto-migration.
// This function is safe to call multiple times; only the first call will take effect.
// The database path is hardcoded to DefaultTaskLogDBPath.
func InitTaskLogDB() error {
	return InitTaskLogDBWithPath(DefaultTaskLogDBPath)
}

// InitTaskLogDBWithPath initializes the task log database with a custom path.
// This is primarily useful for testing or development purposes.
func InitTaskLogDBWithPath(dbPath string) error {
	var initErr error
	taskLogOnce.Do(func() {
		initErr = initTaskLogDB(dbPath)
	})
	return initErr
}

// initTaskLogDB performs the actual task log database initialization.
func initTaskLogDB(dbPath string) error {
	logger.Info("Initializing task log database", zap.String("path", dbPath))

	// Ensure the directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		logger.Error("Failed to create task log db directory", zap.Error(err), zap.String("dir", dbDir))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to create task log db directory", err)
	}

	// Configure GORM logger (silent mode for task logs)
	gormLog := gormlogger.Default.LogMode(gormlogger.Silent)

	// Open SQLite database using the same driver as main database
	dialector := sqlite.Open(dbPath)
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLog,
	})
	if err != nil {
		logger.Error("Failed to open task log database", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to open task log database", err)
	}

	// Apply SQLite optimizations
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to get task log sql.DB", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to get task log sql.DB", err)
	}

	// SQLite connection pool configuration (single connection to avoid concurrent write conflicts)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	// Enable WAL mode (improves concurrent read performance)
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		logger.Warn("Failed to enable WAL mode for task log db", zap.Error(err))
	}

	// Set synchronous=NORMAL (balances performance and safety)
	if err := db.Exec("PRAGMA synchronous = NORMAL").Error; err != nil {
		logger.Warn("Failed to set synchronous mode for task log db", zap.Error(err))
	}

	// Auto-migrate task log model
	if err := db.AutoMigrate(&model.TaskLog{}); err != nil {
		logger.Error("Failed to migrate task log model", zap.Error(err))
		return errors.Wrap(errors.ErrCodeDBMigration, "failed to migrate task log model", err)
	}

	taskLogDB = db

	logger.Info("Task log database initialized successfully",
		zap.String("path", dbPath),
	)

	return nil
}

// GetTaskLogDB returns the task log database connection.
// It panics if the database has not been initialized.
func GetTaskLogDB() *gorm.DB {
	if taskLogDB == nil {
		panic("task log database not initialized - call InitTaskLogDB first")
	}
	return taskLogDB
}

// IsTaskLogDBInitialized returns true if the task log database has been initialized.
func IsTaskLogDBInitialized() bool {
	return taskLogDB != nil
}

// CloseTaskLogDB closes the task log database connection.
func CloseTaskLogDB() error {
	if taskLogDB == nil {
		return nil
	}

	sqlDB, err := taskLogDB.DB()
	if err != nil {
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to get task log sql.DB", err)
	}

	if err := sqlDB.Close(); err != nil {
		return errors.Wrap(errors.ErrCodeDBConnection, "failed to close task log database", err)
	}

	logger.Info("Task log database closed")
	return nil
}

// ResetTaskLogDBForTesting resets the task log database state for testing purposes.
// This allows re-initialization of the database in tests.
// WARNING: Only use this function in tests!
func ResetTaskLogDBForTesting() {
	if taskLogDB != nil {
		sqlDB, _ := taskLogDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		taskLogDB = nil
	}
	taskLogOnce = sync.Once{}
}
