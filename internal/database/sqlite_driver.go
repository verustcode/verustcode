// Package database provides SQLite driver implementation with optimizations.
package database

import (
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/pkg/logger"
)

// SQLiteDriver SQLite数据库驱动
// SQLiteDriver implements the Driver interface for SQLite database
type SQLiteDriver struct{}

// Name 返回驱动名称
// Name returns the driver name
func (d *SQLiteDriver) Name() string {
	return "sqlite"
}

// Open 打开SQLite数据库连接
// Open opens a SQLite database connection
func (d *SQLiteDriver) Open(dsn string) (gorm.Dialector, error) {
	return sqlite.Open(dsn), nil
}

// PreMigrationConfig 在迁移前应用SQLite配置
// PreMigrationConfig applies SQLite configurations before migration
// Note: Foreign key constraints are NOT enabled here to avoid migration failures with orphan records
func (d *SQLiteDriver) PreMigrationConfig(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// SQLite连接池配置（单连接，避免并发写冲突）
	// SQLite connection pool configuration (single connection to avoid concurrent write conflicts)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	// 启用WAL模式（提升并发读性能）
	// Enable WAL mode (improves concurrent read performance)
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		logger.Warn("Failed to enable WAL mode", zap.Error(err))
	}

	// 设置synchronous=NORMAL（平衡性能和安全性）
	// Set synchronous=NORMAL (balances performance and safety)
	if err := db.Exec("PRAGMA synchronous = NORMAL").Error; err != nil {
		logger.Warn("Failed to set synchronous mode", zap.Error(err))
	}

	logger.Info("SQLite pre-migration config applied",
		zap.String("journal_mode", "WAL"),
		zap.String("synchronous", "NORMAL"),
	)

	return nil
}

// PostMigrationConfig 在迁移后应用SQLite配置
// PostMigrationConfig applies SQLite configurations after migration
// Foreign key constraints are enabled here after migration is complete
func (d *SQLiteDriver) PostMigrationConfig(db *gorm.DB) error {
	// 启用外键约束
	// Enable foreign key constraints
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		logger.Warn("Failed to enable foreign keys", zap.Error(err))
	}

	logger.Info("SQLite post-migration config applied",
		zap.Bool("foreign_keys", true),
	)

	return nil
}
