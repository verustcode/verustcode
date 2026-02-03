package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/pkg/logger"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// createInMemoryDB creates an in-memory SQLite database for testing
// This bypasses the global database package to allow independent test databases
func createInMemoryDB(t *testing.T) *gorm.DB {
	// Initialize logger for testing (ignore errors, may already be initialized)
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
		File:   "",
	})

	// Create a unique temporary file-based database for each test
	// Using file-based DB instead of :memory: to avoid connection issues
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	dialector := sqlite.Open(dbPath)
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(model.AllModels()...)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.RemoveAll(tmpDir)
	})

	return db
}

// TestFindPreviousReviewResult tests the FindPreviousReviewResult function
func TestFindPreviousReviewResult(t *testing.T) {
	t.Run("returns empty when db is nil", func(t *testing.T) {
		summary, found := FindPreviousReviewResult(nil, "https://github.com/test/repo/pull/1", "code-quality", "review-1")
		if found {
			t.Error("Expected found to be false when db is nil")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("returns empty when prURL is empty", func(t *testing.T) {
		db := createInMemoryDB(t)

		summary, found := FindPreviousReviewResult(db, "", "code-quality", "review-1")
		if found {
			t.Error("Expected found to be false when prURL is empty")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("returns empty when ruleID is empty", func(t *testing.T) {
		db := createInMemoryDB(t)

		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "", "review-1")
		if found {
			t.Error("Expected found to be false when ruleID is empty")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("returns empty when no previous review exists", func(t *testing.T) {
		db := createInMemoryDB(t)

		// Create current review only
		review := &model.Review{
			ID:      "review-1",
			PRURL:   "https://github.com/test/repo/pull/1",
			RepoURL: "https://github.com/test/repo",
			Status:  model.ReviewStatusCompleted,
		}
		if err := db.Create(review).Error; err != nil {
			t.Fatalf("Failed to create review: %v", err)
		}

		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "code-quality", "review-1")
		if found {
			t.Error("Expected found to be false when no previous review exists")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("finds previous review result", func(t *testing.T) {
		db := createInMemoryDB(t)

		// Create previous review with completed status
		previousReview := &model.Review{
			ID:        "review-1",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-1", // Unique commit SHA
			Status:    model.ReviewStatusCompleted,
		}
		if err := db.Create(previousReview).Error; err != nil {
			t.Fatalf("Failed to create previous review: %v", err)
		}

		// Create previous review rule
		previousRule := &model.ReviewRule{
			ReviewID: "review-1",
			RuleID:   "code-quality",
			Status:   model.RuleStatusCompleted,
		}
		if err := db.Create(previousRule).Error; err != nil {
			t.Fatalf("Failed to create previous rule: %v", err)
		}

		// Create ReviewResult with data (now summary is stored here)
		previousResult := &model.ReviewResult{
			ReviewRuleID: previousRule.ID,
			Data: model.JSONMap{
				"summary": "## Previous Issues\n\n1. SQL injection found\n2. Missing null check",
			},
		}
		if err := db.Create(previousResult).Error; err != nil {
			t.Fatalf("Failed to create previous result: %v", err)
		}

		// Create current review
		currentReview := &model.Review{
			ID:        "review-2",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-2", // Different commit SHA
			Status:    model.ReviewStatusRunning,
		}
		if err := db.Create(currentReview).Error; err != nil {
			t.Fatalf("Failed to create current review: %v", err)
		}

		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "code-quality", "review-2")
		if !found {
			t.Error("Expected found to be true")
		}
		expectedSummary := "## Previous Issues\n\n1. SQL injection found\n2. Missing null check"
		if summary != expectedSummary {
			t.Errorf("Unexpected summary: got %q, want %q", summary, expectedSummary)
		}
	})

	t.Run("does not return incomplete previous review", func(t *testing.T) {
		db := createInMemoryDB(t)

		// Create previous review with running status (not completed)
		previousReview := &model.Review{
			ID:        "review-1",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-1",
			Status:    model.ReviewStatusRunning, // Not completed
		}
		if err := db.Create(previousReview).Error; err != nil {
			t.Fatalf("Failed to create previous review: %v", err)
		}

		// Create current review
		currentReview := &model.Review{
			ID:        "review-2",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-2",
			Status:    model.ReviewStatusRunning,
		}
		if err := db.Create(currentReview).Error; err != nil {
			t.Fatalf("Failed to create current review: %v", err)
		}

		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "code-quality", "review-2")
		if found {
			t.Error("Expected found to be false for incomplete previous review")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("does not return incomplete previous rule", func(t *testing.T) {
		db := createInMemoryDB(t)

		// Create completed previous review
		previousReview := &model.Review{
			ID:        "review-1",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-1",
			Status:    model.ReviewStatusCompleted,
		}
		if err := db.Create(previousReview).Error; err != nil {
			t.Fatalf("Failed to create previous review: %v", err)
		}

		// Create previous review rule with failed status
		previousRule := &model.ReviewRule{
			ReviewID: "review-1",
			RuleID:   "code-quality",
			Status:   model.RuleStatusFailed, // Not completed
		}
		if err := db.Create(previousRule).Error; err != nil {
			t.Fatalf("Failed to create previous rule: %v", err)
		}

		// Create current review
		currentReview := &model.Review{
			ID:        "review-2",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-2",
			Status:    model.ReviewStatusRunning,
		}
		if err := db.Create(currentReview).Error; err != nil {
			t.Fatalf("Failed to create current review: %v", err)
		}

		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "code-quality", "review-2")
		if found {
			t.Error("Expected found to be false for incomplete previous rule")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("returns empty when previous rule has empty summary", func(t *testing.T) {
		db := createInMemoryDB(t)

		// Create completed previous review
		previousReview := &model.Review{
			ID:        "review-1",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-1",
			Status:    model.ReviewStatusCompleted,
		}
		if err := db.Create(previousReview).Error; err != nil {
			t.Fatalf("Failed to create previous review: %v", err)
		}

		// Create previous review rule with no result (no ReviewResult record)
		previousRule := &model.ReviewRule{
			ReviewID: "review-1",
			RuleID:   "code-quality",
			Status:   model.RuleStatusCompleted,
		}
		if err := db.Create(previousRule).Error; err != nil {
			t.Fatalf("Failed to create previous rule: %v", err)
		}
		// Note: No ReviewResult created, so summary should be empty

		// Create current review
		currentReview := &model.Review{
			ID:        "review-2",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-2",
			Status:    model.ReviewStatusRunning,
		}
		if err := db.Create(currentReview).Error; err != nil {
			t.Fatalf("Failed to create current review: %v", err)
		}

		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "code-quality", "review-2")
		if found {
			t.Error("Expected found to be false when summary is empty")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("only matches same rule ID", func(t *testing.T) {
		db := createInMemoryDB(t)

		// Create completed previous review
		previousReview := &model.Review{
			ID:        "review-1",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-1",
			Status:    model.ReviewStatusCompleted,
		}
		if err := db.Create(previousReview).Error; err != nil {
			t.Fatalf("Failed to create previous review: %v", err)
		}

		// Create previous review rule with different rule ID
		previousRule := &model.ReviewRule{
			ReviewID: "review-1",
			RuleID:   "security", // Different rule ID
			Status:   model.RuleStatusCompleted,
		}
		if err := db.Create(previousRule).Error; err != nil {
			t.Fatalf("Failed to create previous rule: %v", err)
		}

		// Create ReviewResult for security rule
		previousResult := &model.ReviewResult{
			ReviewRuleID: previousRule.ID,
			Data: model.JSONMap{
				"summary": "Security issues found",
			},
		}
		if err := db.Create(previousResult).Error; err != nil {
			t.Fatalf("Failed to create previous result: %v", err)
		}

		// Create current review
		currentReview := &model.Review{
			ID:        "review-2",
			PRURL:     "https://github.com/test/repo/pull/1",
			RepoURL:   "https://github.com/test/repo",
			CommitSHA: "commit-2",
			Status:    model.ReviewStatusRunning,
		}
		if err := db.Create(currentReview).Error; err != nil {
			t.Fatalf("Failed to create current review: %v", err)
		}

		// Look for code-quality rule, should not find security rule
		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "code-quality", "review-2")
		if found {
			t.Error("Expected found to be false when rule ID doesn't match")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})

	t.Run("only matches same PR URL", func(t *testing.T) {
		db := createInMemoryDB(t)

		// Create completed previous review for different PR
		previousReview := &model.Review{
			ID:      "review-1",
			PRURL:   "https://github.com/test/repo/pull/99", // Different PR
			RepoURL: "https://github.com/test/repo",
			Status:  model.ReviewStatusCompleted,
		}
		if err := db.Create(previousReview).Error; err != nil {
			t.Fatalf("Failed to create previous review: %v", err)
		}

		previousRule := &model.ReviewRule{
			ReviewID: "review-1",
			RuleID:   "code-quality",
			Status:   model.RuleStatusCompleted,
		}
		if err := db.Create(previousRule).Error; err != nil {
			t.Fatalf("Failed to create previous rule: %v", err)
		}

		// Create ReviewResult
		previousResult := &model.ReviewResult{
			ReviewRuleID: previousRule.ID,
			Data: model.JSONMap{
				"summary": "Issues found in PR 99",
			},
		}
		if err := db.Create(previousResult).Error; err != nil {
			t.Fatalf("Failed to create previous result: %v", err)
		}

		// Create current review for PR 1
		currentReview := &model.Review{
			ID:      "review-2",
			PRURL:   "https://github.com/test/repo/pull/1",
			RepoURL: "https://github.com/test/repo",
			Status:  model.ReviewStatusRunning,
		}
		if err := db.Create(currentReview).Error; err != nil {
			t.Fatalf("Failed to create current review: %v", err)
		}

		// Look for PR 1, should not find PR 99's review
		summary, found := FindPreviousReviewResult(db, "https://github.com/test/repo/pull/1", "code-quality", "review-2")
		if found {
			t.Error("Expected found to be false when PR URL doesn't match")
		}
		if summary != "" {
			t.Errorf("Expected empty summary, got %s", summary)
		}
	})
}
