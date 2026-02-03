// Package model defines the data models for the application.
package model

import (
	"gorm.io/gorm"
)

// EnsureRepositoryConfig ensures a RepositoryReviewConfig record exists for the given repo URL.
// If not exists, creates a new record with default values.
// Returns the config ID and any error encountered.
// Thread-safe: uses GORM's FirstOrCreate which handles concurrent creation gracefully.
func EnsureRepositoryConfig(db *gorm.DB, repoURL string) (uint, error) {
	if repoURL == "" {
		return 0, nil
	}

	config := RepositoryReviewConfig{
		RepoURL: repoURL,
		// ReviewFile and Description are left empty, can be configured later
	}

	// FirstOrCreate: if record exists, does nothing; if not, creates it
	// The unique index on repo_url ensures no duplicates
	result := db.Where("repo_url = ?", repoURL).FirstOrCreate(&config)
	if result.Error != nil {
		return 0, result.Error
	}

	return config.ID, nil
}

