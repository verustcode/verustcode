package store

import (
	"testing"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// TestReportStore_Create tests creating a report
func TestReportStore_Create(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}

	err := store.Report().Create(report)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify the report was created
	retrieved, err := store.Report().GetByID("test-report-001")
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.ID != "test-report-001" {
		t.Errorf("Expected ID 'test-report-001', got '%s'", retrieved.ID)
	}
	if retrieved.RepoURL != "https://github.com/test/repo" {
		t.Errorf("Expected RepoURL 'https://github.com/test/repo', got '%s'", retrieved.RepoURL)
	}
}

// TestReportStore_GetByID tests retrieving a report by ID
func TestReportStore_GetByID(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}
	store.Report().Create(report)

	// Test retrieving existing report
	retrieved, err := store.Report().GetByID(report.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.ID != report.ID {
		t.Errorf("Expected ID '%s', got '%s'", report.ID, retrieved.ID)
	}

	// Test retrieving non-existent report
	_, err = store.Report().GetByID("non-existent")
	if err == nil {
		t.Error("GetByID() should return error for non-existent report")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestReportStore_Update tests updating a report
func TestReportStore_Update(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}
	store.Report().Create(report)

	// Update the report
	report.Status = model.ReportStatusAnalyzing
	report.RepoPath = "/tmp/test-repo"
	err := store.Report().Update(report)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify the update
	retrieved, err := store.Report().GetByID(report.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReportStatusAnalyzing {
		t.Errorf("Expected status 'analyzing', got '%s'", retrieved.Status)
	}
	if retrieved.RepoPath != "/tmp/test-repo" {
		t.Errorf("Expected RepoPath '/tmp/test-repo', got '%s'", retrieved.RepoPath)
	}
}

// TestReportStore_UpdateStatus tests updating report status
func TestReportStore_UpdateStatus(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}
	store.Report().Create(report)

	err := store.Report().UpdateStatus(report.ID, model.ReportStatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	retrieved, err := store.Report().GetByID(report.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReportStatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", retrieved.Status)
	}
}

// TestReportStore_List tests listing reports
func TestReportStore_List(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create multiple reports
	report1 := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo1",
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}
	report2 := &model.Report{
		ID:         "report-002",
		RepoURL:    "https://github.com/test/repo2",
		ReportType: "investment",
		Status:     model.ReportStatusAnalyzing,
	}
	store.Report().Create(report1)
	store.Report().Create(report2)

	// Test listing all reports
	reports, total, err := store.Report().List("", "", 1, 10)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if total < 2 {
		t.Errorf("Expected total >= 2, got %d", total)
	}
	if len(reports) < 2 {
		t.Errorf("Expected at least 2 reports, got %d", len(reports))
	}
}

// TestReportStore_ListByRepository tests listing reports by repository
func TestReportStore_ListByRepository(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	repoURL := "https://github.com/test/repo"

	report1 := &model.Report{
		ID:         "report-001",
		RepoURL:    repoURL,
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}
	report2 := &model.Report{
		ID:         "report-002",
		RepoURL:    repoURL,
		ReportType: "investment",
		Status:     model.ReportStatusAnalyzing,
	}
	store.Report().Create(report1)
	store.Report().Create(report2)

	reports, total, err := store.Report().ListByRepository(repoURL, 10, 0)
	if err != nil {
		t.Fatalf("ListByRepository() failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(reports) != 2 {
		t.Errorf("Expected 2 reports, got %d", len(reports))
	}

	for _, report := range reports {
		if report.RepoURL != repoURL {
			t.Errorf("Expected RepoURL '%s', got '%s'", repoURL, report.RepoURL)
		}
	}
}

// TestReportStore_CreateSection tests creating a report section
func TestReportStore_CreateSection(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}
	store.Report().Create(report)

	section := &model.ReportSection{
		ReportID:     report.ID,
		SectionIndex: 0,
		SectionID:    "section1",
		Status:       model.SectionStatusPending,
	}

	err := store.Report().CreateSection(section)
	if err != nil {
		t.Fatalf("CreateSection() failed: %v", err)
	}

	// Verify the section was created
	sections, err := store.Report().GetSectionsByReportID(report.ID)
	if err != nil {
		t.Fatalf("GetSectionsByReportID() failed: %v", err)
	}

	if len(sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(sections))
	}
	if sections[0].SectionID != "section1" {
		t.Errorf("Expected SectionID 'section1', got '%s'", sections[0].SectionID)
	}
}

// TestReportStore_CountAll tests counting all reports
func TestReportStore_CountAll(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	report1 := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusPending,
	}
	report2 := &model.Report{
		ID:         "report-002",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusAnalyzing,
	}
	store.Report().Create(report1)
	store.Report().Create(report2)

	count, err := store.Report().CountAll()
	if err != nil {
		t.Fatalf("CountAll() failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

// TestReportStore_CancelByID tests canceling a report
func TestReportStore_CancelByID(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "investment",
		Status:     model.ReportStatusAnalyzing,
	}
	store.Report().Create(report)

	rowsAffected, err := store.Report().CancelByID(report.ID)
	if err != nil {
		t.Fatalf("CancelByID() failed: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	// Verify the report was canceled
	retrieved, err := store.Report().GetByID(report.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Status != model.ReportStatusCancelled {
		t.Errorf("Expected status 'cancelled', got '%s'", retrieved.Status)
	}
}

// TestReportStore_ListPendingOrProcessing tests listing pending or processing reports
func TestReportStore_ListPendingOrProcessing(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create reports with different statuses
	reports := []*model.Report{
		{
			ID:         "report-pending",
			RepoURL:    "https://github.com/test/repo1",
			ReportType: "wiki",
			Status:     model.ReportStatusPending,
		},
		{
			ID:         "report-analyzing",
			RepoURL:    "https://github.com/test/repo2",
			ReportType: "wiki",
			Status:     model.ReportStatusAnalyzing,
		},
		{
			ID:         "report-generating",
			RepoURL:    "https://github.com/test/repo3",
			ReportType: "wiki",
			Status:     model.ReportStatusGenerating,
		},
		{
			ID:         "report-completed",
			RepoURL:    "https://github.com/test/repo4",
			ReportType: "wiki",
			Status:     model.ReportStatusCompleted,
		},
		{
			ID:         "report-failed",
			RepoURL:    "https://github.com/test/repo5",
			ReportType: "wiki",
			Status:     model.ReportStatusFailed,
		},
	}

	for _, r := range reports {
		if err := store.Report().Create(r); err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
	}

	// List pending or processing reports
	results, err := store.Report().ListPendingOrProcessing()
	if err != nil {
		t.Fatalf("ListPendingOrProcessing() failed: %v", err)
	}

	// Should return 3 reports (pending, analyzing, generating)
	if len(results) != 3 {
		t.Errorf("Expected 3 reports, got %d", len(results))
	}

	// Verify the correct reports are returned
	statuses := make(map[model.ReportStatus]bool)
	for _, r := range results {
		statuses[r.Status] = true
	}

	if !statuses[model.ReportStatusPending] {
		t.Error("Expected pending report in results")
	}
	if !statuses[model.ReportStatusAnalyzing] {
		t.Error("Expected analyzing report in results")
	}
	if !statuses[model.ReportStatusGenerating] {
		t.Error("Expected generating report in results")
	}
	if statuses[model.ReportStatusCompleted] {
		t.Error("Completed report should not be in results")
	}
	if statuses[model.ReportStatusFailed] {
		t.Error("Failed report should not be in results")
	}
}
