// Package database provides database driver abstraction for extensibility.
// Currently only SQLite is supported, but the interface allows for future
// support of PostgreSQL, MySQL, and other relational databases.
package database

import "gorm.io/gorm"

// Driver 定义数据库驱动接口，用于支持多种数据库
// Driver defines the database driver interface for supporting multiple databases
type Driver interface {
	// Name 返回驱动名称（如 "sqlite", "postgres", "mysql"）
	// Name returns the driver name (e.g., "sqlite", "postgres", "mysql")
	Name() string

	// Open 打开数据库连接
	// Open opens a database connection and returns a GORM dialector
	Open(dsn string) (gorm.Dialector, error)

	// PreMigrationConfig 在迁移前应用数据库配置（连接池、WAL模式等）
	// PreMigrationConfig applies database configurations before migration (connection pool, WAL mode, etc.)
	// Note: Foreign key constraints should NOT be enabled here to avoid migration failures
	PreMigrationConfig(db *gorm.DB) error

	// PostMigrationConfig 在迁移后应用数据库配置（外键约束等）
	// PostMigrationConfig applies database configurations after migration (foreign key constraints, etc.)
	PostMigrationConfig(db *gorm.DB) error
}
