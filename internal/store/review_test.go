package store

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// TestReviewStore_Create tests creating a review
func TestReviewStore_Create(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := &model.Review{
		ID:        "test-review-001",
		Ref:       "main",
		CommitSHA: "abc123def456",
		RepoURL:   "https://github.com/test/repo",
		Source:    "cli",
		Status:    model.ReviewStatusPending,
	}

	err := store.Review().Create(review)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify the review was created
	retrieved, err := store.Review().GetByID("test-review-001")
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.ID != "test-review-001" {
		t.Errorf("Expected ID 'test-review-001', got '%s'", retrieved.ID)
	}
	if retrieved.RepoURL != "https://github.com/test/repo" {
		t.Errorf("Expected RepoURL 'https://github.com/test/repo', got '%s'", retrieved.RepoURL)
	}
}

// TestReviewStore_GetByID tests retrieving a review by ID
func TestReviewStore_GetByID(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create a review first
	review := CreateTestReview(t, store)

	// Test retrieving existing review
	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.ID != review.ID {
		t.Errorf("Expected ID '%s', got '%s'", review.ID, retrieved.ID)
	}

	// Test retrieving non-existent review
	_, err = store.Review().GetByID("non-existent")
	if err == nil {
		t.Error("GetByID() should return error for non-existent review")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestReviewStore_GetByIDWithDetails tests retrieving a review with all details
func TestReviewStore_GetByIDWithDetails(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create a review with rules
	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)

	// Create a run for the rule
	run := &model.ReviewRuleRun{
		ReviewRuleID: rule.ID,
		RunIndex:     0,
		Agent:        "test-agent",
		Status:       model.RunStatusCompleted,
	}
	if err := store.Review().CreateRun(run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}

	// Create a result for the rule
	result := &model.ReviewResult{
		ReviewRuleID: rule.ID,
		Data: model.JSONMap{
			"findings": []interface{}{"finding1", "finding2"},
		},
	}
	if err := store.Review().CreateResult(result); err != nil {
		t.Fatalf("Failed to create result: %v", err)
	}

	// Retrieve with details
	retrieved, err := store.Review().GetByIDWithDetails(review.ID)
	if err != nil {
		t.Fatalf("GetByIDWithDetails() failed: %v", err)
	}

	if len(retrieved.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(retrieved.Rules))
	}
	if len(retrieved.Rules[0].Runs) != 1 {
		t.Errorf("Expected 1 run, got %d", len(retrieved.Rules[0].Runs))
	}
	if len(retrieved.Rules[0].Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(retrieved.Rules[0].Results))
	}
}

// TestReviewStore_GetByIDWithRules tests retrieving a review with rules
func TestReviewStore_GetByIDWithRules(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	CreateTestReviewRule(t, store, review.ID)
	CreateTestReviewRule(t, store, review.ID, func(r *model.ReviewRule) {
		r.RuleIndex = 1
		r.RuleID = "test-rule-002"
	})

	retrieved, err := store.Review().GetByIDWithRules(review.ID)
	if err != nil {
		t.Fatalf("GetByIDWithRules() failed: %v", err)
	}

	if len(retrieved.Rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(retrieved.Rules))
	}
}

// TestReviewStore_Update tests updating a review
func TestReviewStore_Update(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	// Update the review
	review.Status = model.ReviewStatusRunning
	review.RepoPath = "/tmp/test-repo"
	err := store.Review().Update(review)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify the update
	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReviewStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", retrieved.Status)
	}
	if retrieved.RepoPath != "/tmp/test-repo" {
		t.Errorf("Expected RepoPath '/tmp/test-repo', got '%s'", retrieved.RepoPath)
	}
}

// TestReviewStore_Save tests saving a review
func TestReviewStore_Save(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	// Modify and save
	review.Status = model.ReviewStatusCompleted
	now := time.Now()
	review.CompletedAt = &now
	err := store.Review().Save(review)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify the save
	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReviewStatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", retrieved.Status)
	}
	if retrieved.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

// TestReviewStore_Delete tests deleting a review
func TestReviewStore_Delete(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	// Delete the review
	err := store.Review().Delete(review.ID)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify it's deleted (soft delete)
	_, err = store.Review().GetByID(review.ID)
	if err == nil {
		t.Error("GetByID() should return error after delete")
	}
}

// TestReviewStore_UpdateStatus tests updating review status
func TestReviewStore_UpdateStatus(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	err := store.Review().UpdateStatus(review.ID, model.ReviewStatusRunning)
	if err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReviewStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", retrieved.Status)
	}
}

// TestReviewStore_UpdateStatusWithError tests updating status with error message
func TestReviewStore_UpdateStatusWithError(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	errMsg := "test error message"

	err := store.Review().UpdateStatusWithError(review.ID, model.ReviewStatusFailed, errMsg)
	if err != nil {
		t.Fatalf("UpdateStatusWithError() failed: %v", err)
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReviewStatusFailed {
		t.Errorf("Expected status 'failed', got '%s'", retrieved.Status)
	}
	if retrieved.ErrorMessage != errMsg {
		t.Errorf("Expected error message '%s', got '%s'", errMsg, retrieved.ErrorMessage)
	}
}

// TestReviewStore_UpdateStatusToRunningIfPending tests conditional status update
func TestReviewStore_UpdateStatusToRunningIfPending(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	startedAt := time.Now()

	// Test updating from pending to running
	updated, err := store.Review().UpdateStatusToRunningIfPending(review.ID, startedAt)
	if err != nil {
		t.Fatalf("UpdateStatusToRunningIfPending() failed: %v", err)
	}
	if !updated {
		t.Error("Expected update to succeed")
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReviewStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", retrieved.Status)
	}
	if retrieved.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}

	// Test updating from running (should still work)
	updated, err = store.Review().UpdateStatusToRunningIfPending(review.ID, startedAt)
	if err != nil {
		t.Fatalf("UpdateStatusToRunningIfPending() failed: %v", err)
	}
	if !updated {
		t.Error("Expected update to succeed for running status")
	}

	// Test updating from completed (should fail)
	store.Review().UpdateStatus(review.ID, model.ReviewStatusCompleted)
	updated, err = store.Review().UpdateStatusToRunningIfPending(review.ID, startedAt)
	if err != nil {
		t.Fatalf("UpdateStatusToRunningIfPending() failed: %v", err)
	}
	if updated {
		t.Error("Expected update to fail for completed status")
	}
}

// TestReviewStore_UpdateProgress tests updating progress
func TestReviewStore_UpdateProgress(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	err := store.Review().UpdateProgress(review.ID, 5)
	if err != nil {
		t.Fatalf("UpdateProgress() failed: %v", err)
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.CurrentRuleIndex != 5 {
		t.Errorf("Expected CurrentRuleIndex 5, got %d", retrieved.CurrentRuleIndex)
	}
}

// TestReviewStore_UpdateCurrentRuleIndex tests updating current rule index
func TestReviewStore_UpdateCurrentRuleIndex(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	err := store.Review().UpdateCurrentRuleIndex(review.ID, 10)
	if err != nil {
		t.Fatalf("UpdateCurrentRuleIndex() failed: %v", err)
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.CurrentRuleIndex != 10 {
		t.Errorf("Expected CurrentRuleIndex 10, got %d", retrieved.CurrentRuleIndex)
	}
}

// TestReviewStore_UpdateRepoPath tests updating repo path
func TestReviewStore_UpdateRepoPath(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	newPath := "/tmp/new-repo-path"

	err := store.Review().UpdateRepoPath(review.ID, newPath)
	if err != nil {
		t.Fatalf("UpdateRepoPath() failed: %v", err)
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.RepoPath != newPath {
		t.Errorf("Expected RepoPath '%s', got '%s'", newPath, retrieved.RepoPath)
	}
}

// TestReviewStore_UpdateMetadata tests updating metadata
func TestReviewStore_UpdateMetadata(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	updates := map[string]interface{}{
		"lines_added":   100,
		"lines_deleted": 50,
		"files_changed": 10,
	}

	err := store.Review().UpdateMetadata(review.ID, updates)
	if err != nil {
		t.Fatalf("UpdateMetadata() failed: %v", err)
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.LinesAdded != 100 {
		t.Errorf("Expected LinesAdded 100, got %d", retrieved.LinesAdded)
	}
	if retrieved.LinesDeleted != 50 {
		t.Errorf("Expected LinesDeleted 50, got %d", retrieved.LinesDeleted)
	}
	if retrieved.FilesChanged != 10 {
		t.Errorf("Expected FilesChanged 10, got %d", retrieved.FilesChanged)
	}
}

// TestReviewStore_IncrementRetryCount tests incrementing retry count
func TestReviewStore_IncrementRetryCount(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	// Increment retry count
	err := store.Review().IncrementRetryCount(review.ID)
	if err != nil {
		t.Fatalf("IncrementRetryCount() failed: %v", err)
	}

	retrieved, err := store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.RetryCount != 1 {
		t.Errorf("Expected RetryCount 1, got %d", retrieved.RetryCount)
	}

	// Increment again
	err = store.Review().IncrementRetryCount(review.ID)
	if err != nil {
		t.Fatalf("IncrementRetryCount() failed: %v", err)
	}

	retrieved, err = store.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.RetryCount != 2 {
		t.Errorf("Expected RetryCount 2, got %d", retrieved.RetryCount)
	}
}

// TestReviewStore_UpdateStatusIfAllowed tests conditional status update
func TestReviewStore_UpdateStatusIfAllowed(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)

	// Test allowed status update
	allowedStatuses := []model.ReviewStatus{model.ReviewStatusPending, model.ReviewStatusRunning}
	rowsAffected, err := store.Review().UpdateStatusIfAllowed(review.ID, model.ReviewStatusRunning, allowedStatuses)
	if err != nil {
		t.Fatalf("UpdateStatusIfAllowed() failed: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	// Test disallowed status update - review is now Running, not in allowedStatuses for Completed
	// Update allowedStatuses to only include Pending (not Running)
	disallowedStatuses := []model.ReviewStatus{model.ReviewStatusPending}
	rowsAffected, err = store.Review().UpdateStatusIfAllowed(review.ID, model.ReviewStatusCompleted, disallowedStatuses)
	if err != nil {
		t.Fatalf("UpdateStatusIfAllowed() failed: %v", err)
	}
	if rowsAffected != 0 {
		t.Errorf("Expected 0 rows affected, got %d", rowsAffected)
	}
}

// TestReviewStore_List tests listing reviews
func TestReviewStore_List(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create multiple reviews
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.Status = model.ReviewStatusPending
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.Status = model.ReviewStatusRunning
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-003"
		r.Status = model.ReviewStatusCompleted
	})

	// Test listing all reviews
	reviews, total, err := store.Review().List("", 10, 0)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
	if len(reviews) != 3 {
		t.Errorf("Expected 3 reviews, got %d", len(reviews))
	}

	// Test filtering by status
	reviews, total, err = store.Review().List(string(model.ReviewStatusPending), 10, 0)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}
	if len(reviews) != 1 {
		t.Errorf("Expected 1 review, got %d", len(reviews))
	}
	if reviews[0].ID != "review-001" {
		t.Errorf("Expected review ID 'review-001', got '%s'", reviews[0].ID)
	}

	// Test pagination
	reviews, total, err = store.Review().List("", 2, 0)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(reviews) != 2 {
		t.Errorf("Expected 2 reviews, got %d", len(reviews))
	}
}

// TestReviewStore_ListByRepository tests listing reviews by repository
func TestReviewStore_ListByRepository(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	repoURL1 := "https://github.com/test/repo1"
	repoURL2 := "https://github.com/test/repo2"

	// Create reviews for different repositories
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.RepoURL = repoURL1
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.RepoURL = repoURL1
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-003"
		r.RepoURL = repoURL2
	})

	// Test listing by repository
	reviews, total, err := store.Review().ListByRepository(repoURL1, 10, 0)
	if err != nil {
		t.Fatalf("ListByRepository() failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(reviews) != 2 {
		t.Errorf("Expected 2 reviews, got %d", len(reviews))
	}

	for _, review := range reviews {
		if review.RepoURL != repoURL1 {
			t.Errorf("Expected RepoURL '%s', got '%s'", repoURL1, review.RepoURL)
		}
	}
}

// TestReviewStore_ListByStatus tests listing reviews by status
func TestReviewStore_ListByStatus(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create reviews with different statuses
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.Status = model.ReviewStatusPending
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.Status = model.ReviewStatusPending
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-003"
		r.Status = model.ReviewStatusRunning
	})

	// Test listing by status
	reviews, err := store.Review().ListByStatus(model.ReviewStatusPending)
	if err != nil {
		t.Fatalf("ListByStatus() failed: %v", err)
	}

	if len(reviews) != 2 {
		t.Errorf("Expected 2 reviews, got %d", len(reviews))
	}

	for _, review := range reviews {
		if review.Status != model.ReviewStatusPending {
			t.Errorf("Expected status 'pending', got '%s'", review.Status)
		}
	}
}

// TestReviewStore_ListPendingOrRunning tests listing pending or running reviews
func TestReviewStore_ListPendingOrRunning(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create reviews with different statuses
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.Status = model.ReviewStatusPending
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.Status = model.ReviewStatusRunning
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-003"
		r.Status = model.ReviewStatusCompleted
	})

	// Test listing pending or running
	reviews, err := store.Review().ListPendingOrRunning()
	if err != nil {
		t.Fatalf("ListPendingOrRunning() failed: %v", err)
	}

	if len(reviews) != 2 {
		t.Errorf("Expected 2 reviews, got %d", len(reviews))
	}

	for _, review := range reviews {
		if review.Status != model.ReviewStatusPending && review.Status != model.ReviewStatusRunning {
			t.Errorf("Expected status 'pending' or 'running', got '%s'", review.Status)
		}
	}
}

// TestReviewStore_GetByPRURLAndCommit tests retrieving review by PR URL and commit
func TestReviewStore_GetByPRURLAndCommit(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	prURL := "https://github.com/test/repo/pull/123"
	commitSHA := "abc123def456"

	review := CreateTestReview(t, store, func(r *model.Review) {
		r.PRURL = prURL
		r.CommitSHA = commitSHA
	})

	// Test retrieving by PR URL and commit
	retrieved, err := store.Review().GetByPRURLAndCommit(prURL, commitSHA)
	if err != nil {
		t.Fatalf("GetByPRURLAndCommit() failed: %v", err)
	}

	if retrieved.ID != review.ID {
		t.Errorf("Expected ID '%s', got '%s'", review.ID, retrieved.ID)
	}

	// Test non-existent combination
	_, err = store.Review().GetByPRURLAndCommit(prURL, "non-existent")
	if err == nil {
		t.Error("GetByPRURLAndCommit() should return error for non-existent commit")
	}
}

// TestReviewStore_CreateRule tests creating a review rule
func TestReviewStore_CreateRule(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)

	// Verify the rule was created
	retrieved, err := store.Review().GetRuleByID(rule.ID)
	if err != nil {
		t.Fatalf("GetRuleByID() failed: %v", err)
	}

	if retrieved.ReviewID != review.ID {
		t.Errorf("Expected ReviewID '%s', got '%s'", review.ID, retrieved.ReviewID)
	}
	if retrieved.RuleID != "test-rule-001" {
		t.Errorf("Expected RuleID 'test-rule-001', got '%s'", retrieved.RuleID)
	}
}

// TestReviewStore_BatchCreateRules tests batch creating rules
func TestReviewStore_BatchCreateRules(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rules := []model.ReviewRule{
		{
			ReviewID:  review.ID,
			RuleIndex: 0,
			RuleID:    "rule-001",
			Status:    model.RuleStatusPending,
		},
		{
			ReviewID:  review.ID,
			RuleIndex: 1,
			RuleID:    "rule-002",
			Status:    model.RuleStatusPending,
		},
		{
			ReviewID:  review.ID,
			RuleIndex: 2,
			RuleID:    "rule-003",
			Status:    model.RuleStatusPending,
		},
	}

	err := store.Review().BatchCreateRules(rules)
	if err != nil {
		t.Fatalf("BatchCreateRules() failed: %v", err)
	}

	// Verify all rules were created
	retrievedRules, err := store.Review().GetRulesByReviewID(review.ID)
	if err != nil {
		t.Fatalf("GetRulesByReviewID() failed: %v", err)
	}

	if len(retrievedRules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(retrievedRules))
	}
}

// TestReviewStore_GetRulesByReviewID tests retrieving rules by review ID
func TestReviewStore_GetRulesByReviewID(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	CreateTestReviewRule(t, store, review.ID, func(r *model.ReviewRule) {
		r.RuleIndex = 0
		r.RuleID = "rule-001"
	})
	CreateTestReviewRule(t, store, review.ID, func(r *model.ReviewRule) {
		r.RuleIndex = 1
		r.RuleID = "rule-002"
	})

	rules, err := store.Review().GetRulesByReviewID(review.ID)
	if err != nil {
		t.Fatalf("GetRulesByReviewID() failed: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	// Verify ordering
	if rules[0].RuleIndex != 0 {
		t.Errorf("Expected first rule index 0, got %d", rules[0].RuleIndex)
	}
	if rules[1].RuleIndex != 1 {
		t.Errorf("Expected second rule index 1, got %d", rules[1].RuleIndex)
	}
}

// TestReviewStore_UpdateRule tests updating a rule
func TestReviewStore_UpdateRule(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)

	// Update the rule
	rule.Status = model.RuleStatusCompleted
	rule.FindingsCount = 5
	err := store.Review().UpdateRule(rule)
	if err != nil {
		t.Fatalf("UpdateRule() failed: %v", err)
	}

	// Verify the update
	retrieved, err := store.Review().GetRuleByID(rule.ID)
	if err != nil {
		t.Fatalf("GetRuleByID() failed: %v", err)
	}

	if retrieved.Status != model.RuleStatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", retrieved.Status)
	}
	if retrieved.FindingsCount != 5 {
		t.Errorf("Expected findings_count 5, got %d", retrieved.FindingsCount)
	}
}

// TestReviewStore_UpdateRuleStatus tests updating rule status
func TestReviewStore_UpdateRuleStatus(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)

	err := store.Review().UpdateRuleStatus(rule.ID, model.RuleStatusRunning)
	if err != nil {
		t.Fatalf("UpdateRuleStatus() failed: %v", err)
	}

	retrieved, err := store.Review().GetRuleByID(rule.ID)
	if err != nil {
		t.Fatalf("GetRuleByID() failed: %v", err)
	}

	if retrieved.Status != model.RuleStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", retrieved.Status)
	}
}

// TestReviewStore_UpdateRuleStatusWithError tests updating rule status with error
func TestReviewStore_UpdateRuleStatusWithError(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)
	errMsg := "rule execution failed"

	err := store.Review().UpdateRuleStatusWithError(rule.ID, model.RuleStatusFailed, errMsg)
	if err != nil {
		t.Fatalf("UpdateRuleStatusWithError() failed: %v", err)
	}

	retrieved, err := store.Review().GetRuleByID(rule.ID)
	if err != nil {
		t.Fatalf("GetRuleByID() failed: %v", err)
	}

	if retrieved.Status != model.RuleStatusFailed {
		t.Errorf("Expected status 'failed', got '%s'", retrieved.Status)
	}
	if retrieved.ErrorMessage != errMsg {
		t.Errorf("Expected error message '%s', got '%s'", errMsg, retrieved.ErrorMessage)
	}
}

// TestReviewStore_CreateRun tests creating a review rule run
func TestReviewStore_CreateRun(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)

	run := &model.ReviewRuleRun{
		ReviewRuleID: rule.ID,
		RunIndex:     0,
		Agent:        "test-agent",
		Status:       model.RunStatusPending,
	}

	err := store.Review().CreateRun(run)
	if err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	// Verify the run was created
	retrieved, err := store.Review().GetRunByID(run.ID)
	if err != nil {
		t.Fatalf("GetRunByID() failed: %v", err)
	}

	if retrieved.ReviewRuleID != rule.ID {
		t.Errorf("Expected ReviewRuleID %d, got %d", rule.ID, retrieved.ReviewRuleID)
	}
	if retrieved.Agent != "test-agent" {
		t.Errorf("Expected Agent 'test-agent', got '%s'", retrieved.Agent)
	}
}

// TestReviewStore_GetRunsByRuleID tests retrieving runs by rule ID
func TestReviewStore_GetRunsByRuleID(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)

	// Create multiple runs
	run1 := &model.ReviewRuleRun{
		ReviewRuleID: rule.ID,
		RunIndex:     0,
		Agent:        "agent-1",
		Status:       model.RunStatusCompleted,
	}
	run2 := &model.ReviewRuleRun{
		ReviewRuleID: rule.ID,
		RunIndex:     1,
		Agent:        "agent-2",
		Status:       model.RunStatusPending,
	}

	store.Review().CreateRun(run1)
	store.Review().CreateRun(run2)

	runs, err := store.Review().GetRunsByRuleID(rule.ID)
	if err != nil {
		t.Fatalf("GetRunsByRuleID() failed: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("Expected 2 runs, got %d", len(runs))
	}

	// Verify ordering
	if runs[0].RunIndex != 0 {
		t.Errorf("Expected first run index 0, got %d", runs[0].RunIndex)
	}
	if runs[1].RunIndex != 1 {
		t.Errorf("Expected second run index 1, got %d", runs[1].RunIndex)
	}
}

// TestReviewStore_UpdateRun tests updating a run
func TestReviewStore_UpdateRun(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)
	run := &model.ReviewRuleRun{
		ReviewRuleID: rule.ID,
		RunIndex:     0,
		Agent:        "test-agent",
		Status:       model.RunStatusPending,
	}
	store.Review().CreateRun(run)

	// Update the run
	run.Status = model.RunStatusCompleted
	run.FindingsCount = 3
	now := time.Now()
	run.CompletedAt = &now
	err := store.Review().UpdateRun(run)
	if err != nil {
		t.Fatalf("UpdateRun() failed: %v", err)
	}

	// Verify the update
	retrieved, err := store.Review().GetRunByID(run.ID)
	if err != nil {
		t.Fatalf("GetRunByID() failed: %v", err)
	}

	if retrieved.Status != model.RunStatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", retrieved.Status)
	}
	if retrieved.FindingsCount != 3 {
		t.Errorf("Expected findings_count 3, got %d", retrieved.FindingsCount)
	}
}

// TestReviewStore_CreateResult tests creating a review result
func TestReviewStore_CreateResult(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule := CreateTestReviewRule(t, store, review.ID)

	result := &model.ReviewResult{
		ReviewRuleID: rule.ID,
		Data: model.JSONMap{
			"findings": []interface{}{
				map[string]interface{}{
					"severity": "high",
					"message":  "test finding",
				},
			},
		},
	}

	err := store.Review().CreateResult(result)
	if err != nil {
		t.Fatalf("CreateResult() failed: %v", err)
	}

	// Verify the result was created
	results, err := store.Review().GetResultsByRuleID(rule.ID)
	if err != nil {
		t.Fatalf("GetResultsByRuleID() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].ReviewRuleID != rule.ID {
		t.Errorf("Expected ReviewRuleID %d, got %d", rule.ID, results[0].ReviewRuleID)
	}
}

// TestReviewStore_GetResultsByReviewID tests retrieving results by review ID
func TestReviewStore_GetResultsByReviewID(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	review := CreateTestReview(t, store)
	rule1 := CreateTestReviewRule(t, store, review.ID, func(r *model.ReviewRule) {
		r.RuleIndex = 0
		r.RuleID = "rule-001"
	})
	rule2 := CreateTestReviewRule(t, store, review.ID, func(r *model.ReviewRule) {
		r.RuleIndex = 1
		r.RuleID = "rule-002"
	})

	// Create results for both rules
	result1 := &model.ReviewResult{
		ReviewRuleID: rule1.ID,
		Data:         model.JSONMap{"finding": "result1"},
	}
	result2 := &model.ReviewResult{
		ReviewRuleID: rule2.ID,
		Data:         model.JSONMap{"finding": "result2"},
	}

	store.Review().CreateResult(result1)
	store.Review().CreateResult(result2)

	// Retrieve results by review ID
	results, err := store.Review().GetResultsByReviewID(review.ID)
	if err != nil {
		t.Fatalf("GetResultsByReviewID() failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

// TestReviewStore_CountAll tests counting all reviews
func TestReviewStore_CountAll(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create multiple reviews
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-003"
	})

	count, err := store.Review().CountAll()
	if err != nil {
		t.Fatalf("CountAll() failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

// TestReviewStore_CountByStatusOnly tests counting reviews by status
func TestReviewStore_CountByStatusOnly(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create reviews with different statuses
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.Status = model.ReviewStatusPending
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.Status = model.ReviewStatusPending
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-003"
		r.Status = model.ReviewStatusRunning
	})

	count, err := store.Review().CountByStatusOnly(model.ReviewStatusPending)
	if err != nil {
		t.Fatalf("CountByStatusOnly() failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

// TestReviewStore_CountByStatusAndDateRange tests counting reviews by status and date range
func TestReviewStore_CountByStatusAndDateRange(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// Create reviews at different times
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.Status = model.ReviewStatusCompleted
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.Status = model.ReviewStatusCompleted
	})

	// Note: GORM sets CreatedAt automatically, so we test with current time range
	// Use a wide date range that includes both reviews
	start := yesterday.Add(-24 * time.Hour)
	end := now.Add(24 * time.Hour)

	count, err := store.Review().CountByStatusAndDateRange(model.ReviewStatusCompleted, start, end)
	if err != nil {
		t.Fatalf("CountByStatusAndDateRange() failed: %v", err)
	}

	if count < 2 {
		t.Errorf("Expected count >= 2, got %d", count)
	}
}

// TestReviewStore_GetMaxRevisionByPRURL tests getting max revision by PR URL
func TestReviewStore_GetMaxRevisionByPRURL(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	prURL := "https://github.com/test/repo/pull/123"

	// Create reviews with different revision counts
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.PRURL = prURL
		r.RevisionCount = 1
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.PRURL = prURL
		r.RevisionCount = 3
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-003"
		r.PRURL = prURL
		r.RevisionCount = 2
	})

	maxRevision, err := store.Review().GetMaxRevisionByPRURL(prURL)
	if err != nil {
		t.Fatalf("GetMaxRevisionByPRURL() failed: %v", err)
	}

	if maxRevision != 3 {
		t.Errorf("Expected max revision 3, got %d", maxRevision)
	}
}

// TestReviewStore_UpdateMergedAtByPRURL tests updating merged_at by PR URL
func TestReviewStore_UpdateMergedAtByPRURL(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	prURL := "https://github.com/test/repo/pull/123"
	mergedAt := time.Now()

	// Create reviews with the same PR URL
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-001"
		r.PRURL = prURL
	})
	CreateTestReview(t, store, func(r *model.Review) {
		r.ID = "review-002"
		r.PRURL = prURL
	})

	rowsAffected, err := store.Review().UpdateMergedAtByPRURL(prURL, mergedAt)
	if err != nil {
		t.Fatalf("UpdateMergedAtByPRURL() failed: %v", err)
	}

	if rowsAffected != 2 {
		t.Errorf("Expected 2 rows affected, got %d", rowsAffected)
	}

	// Verify the update
	reviews, _, err := store.Review().ListByRepository("", 10, 0)
	if err != nil {
		t.Fatalf("ListByRepository() failed: %v", err)
	}

	for _, review := range reviews {
		if review.PRURL == prURL && review.MergedAt == nil {
			t.Error("Expected MergedAt to be set for reviews with PR URL")
		}
	}
}
