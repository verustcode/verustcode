// Package store provides test utilities for database testing.
package store

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/database"
	"github.com/verustcode/verustcode/internal/model"
)

// SetupTestDB creates an in-memory SQLite database for testing.
// It returns a Store instance and a cleanup function.
// The cleanup function should be called with defer in tests.
func SetupTestDB(t *testing.T) (Store, func()) {
	// Reset database state to allow re-initialization
	database.ResetForTesting()

	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Initialize database with temp path
	if err := database.InitWithPath(tmpPath); err != nil {
		os.Remove(tmpPath)
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	db := database.Get()
	store := NewStore(db)

	// Cleanup function
	cleanup := func() {
		database.Close()
		database.ResetForTesting()
		os.Remove(tmpPath)
	}

	return store, cleanup
}

// SetupTestDBWithModels creates an in-memory SQLite database and runs migrations.
// This is a convenience function that ensures all models are migrated.
func SetupTestDBWithModels(t *testing.T) (*gorm.DB, func()) {
	// Reset database state
	database.ResetForTesting()

	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Initialize database
	if err := database.InitWithPath(tmpPath); err != nil {
		os.Remove(tmpPath)
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	db := database.Get()

	// Ensure all models are migrated
	models := model.AllModels()
	if err := db.AutoMigrate(models...); err != nil {
		database.Close()
		database.ResetForTesting()
		os.Remove(tmpPath)
		t.Fatalf("Failed to migrate models: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		database.Close()
		database.ResetForTesting()
		os.Remove(tmpPath)
	}

	return db, cleanup
}

// CreateTestReview creates a test Review with default values.
// Fields can be overridden by passing a function that modifies the review.
func CreateTestReview(t *testing.T, store Store, overrides ...func(*model.Review)) *model.Review {
	// Generate unique values to avoid UNIQUE constraint violations
	uniqueID := t.Name() + "-" + time.Now().Format("150405.000000")
	uniqueCommitSHA := fmt.Sprintf("%x", sha256.Sum256([]byte(uniqueID)))[:40]

	review := &model.Review{
		ID:        uniqueID,
		Ref:       "main",
		CommitSHA: uniqueCommitSHA,
		RepoURL:   "https://github.com/test/repo",
		Source:    "cli",
		Status:    model.ReviewStatusPending,
	}

	// Apply overrides
	for _, override := range overrides {
		override(review)
	}

	if err := store.Review().Create(review); err != nil {
		t.Fatalf("Failed to create test review: %v", err)
	}

	return review
}

// CreateTestReviewRule creates a test ReviewRule with default values.
func CreateTestReviewRule(t *testing.T, store Store, reviewID string, overrides ...func(*model.ReviewRule)) *model.ReviewRule {
	rule := &model.ReviewRule{
		ReviewID:  reviewID,
		RuleIndex: 0,
		RuleID:    "test-rule-001",
		Status:    model.RuleStatusPending,
		RuleConfig: model.JSONMap{
			"name": "test rule",
		},
	}

	// Apply overrides
	for _, override := range overrides {
		override(rule)
	}

	if err := store.Review().CreateRule(rule); err != nil {
		t.Fatalf("Failed to create test review rule: %v", err)
	}

	return rule
}
