// Package store provides data access layer interfaces and implementations.
// This package abstracts database operations to improve maintainability
// and decouple business logic from specific database implementations.
package store

import "gorm.io/gorm"

// Store aggregates all data store interfaces.
// It provides a single point of access for all database operations.
type Store interface {
	Review() ReviewStore
	Report() ReportStore
	Settings() SettingsStore
	RepositoryConfig() RepositoryConfigStore

	// DB returns the underlying database connection for advanced operations.
	// Use sparingly - prefer using specific store methods.
	DB() *gorm.DB

	// Transaction executes operations within a database transaction.
	Transaction(fn func(Store) error) error
}

// gormStore implements Store interface using GORM.
type gormStore struct {
	db               *gorm.DB
	reviewStore      ReviewStore
	reportStore      ReportStore
	settingsStore    SettingsStore
	repoConfigStore  RepositoryConfigStore
}

// NewStore creates a new Store instance with GORM backend.
func NewStore(db *gorm.DB) Store {
	return &gormStore{
		db:               db,
		reviewStore:      newReviewStore(db),
		reportStore:      newReportStore(db),
		settingsStore:    newSettingsStore(db),
		repoConfigStore:  newRepositoryConfigStore(db),
	}
}

func (s *gormStore) Review() ReviewStore {
	return s.reviewStore
}

func (s *gormStore) Report() ReportStore {
	return s.reportStore
}

func (s *gormStore) Settings() SettingsStore {
	return s.settingsStore
}

func (s *gormStore) RepositoryConfig() RepositoryConfigStore {
	return s.repoConfigStore
}

func (s *gormStore) DB() *gorm.DB {
	return s.db
}

func (s *gormStore) Transaction(fn func(Store) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txStore := &gormStore{
			db:               tx,
			reviewStore:      newReviewStore(tx),
			reportStore:      newReportStore(tx),
			settingsStore:    newSettingsStore(tx),
			repoConfigStore:  newRepositoryConfigStore(tx),
		}
		return fn(txStore)
	})
}

