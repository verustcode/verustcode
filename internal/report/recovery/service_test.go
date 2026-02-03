package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

func init() {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
}

// mockTaskEnqueuer implements TaskEnqueuer for testing
type mockTaskEnqueuer struct {
	enqueuedReports []*model.Report
	enqueueReturn   bool
}

func (m *mockTaskEnqueuer) Enqueue(report *model.Report, callback func(*model.Report, error)) bool {
	m.enqueuedReports = append(m.enqueuedReports, report)
	return m.enqueueReturn
}

// TestRecoverToQueue_NoReports tests recovery when there are no pending reports
func TestRecoverToQueue_NoReports(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: true}
	service := NewService(cfg, s, mockEnqueuer)

	// Should not panic and should log "No pending reports to recover"
	service.RecoverToQueue(context.Background())

	assert.Empty(t, mockEnqueuer.enqueuedReports)
}

// TestRecoverToQueue_PendingReport tests recovery of a pending report
func TestRecoverToQueue_PendingReport(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	// Create a pending report
	report := &model.Report{
		ID:         "test-report-1",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}
	require.NoError(t, s.Report().Create(report))

	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: true}
	service := NewService(cfg, s, mockEnqueuer)

	service.RecoverToQueue(context.Background())

	// Should enqueue the report
	assert.Len(t, mockEnqueuer.enqueuedReports, 1)
	assert.Equal(t, "test-report-1", mockEnqueuer.enqueuedReports[0].ID)
}

// TestRecoverToQueue_AnalyzingReport tests recovery of an analyzing report
func TestRecoverToQueue_AnalyzingReport(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	startedAt := time.Now().Add(-1 * time.Hour)
	report := &model.Report{
		ID:         "test-report-2",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusAnalyzing,
		StartedAt:  &startedAt,
	}
	require.NoError(t, s.Report().Create(report))

	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: true}
	service := NewService(cfg, s, mockEnqueuer)

	service.RecoverToQueue(context.Background())

	// Should enqueue the report
	assert.Len(t, mockEnqueuer.enqueuedReports, 1)
	assert.Equal(t, "test-report-2", mockEnqueuer.enqueuedReports[0].ID)
}

// TestRecoverToQueue_GeneratingReport tests recovery of a generating report
func TestRecoverToQueue_GeneratingReport(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	startedAt := time.Now().Add(-2 * time.Hour)
	report := &model.Report{
		ID:         "test-report-3",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusGenerating,
		StartedAt:  &startedAt,
	}
	require.NoError(t, s.Report().Create(report))

	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: true}
	service := NewService(cfg, s, mockEnqueuer)

	service.RecoverToQueue(context.Background())

	// Should enqueue the report
	assert.Len(t, mockEnqueuer.enqueuedReports, 1)
	assert.Equal(t, "test-report-3", mockEnqueuer.enqueuedReports[0].ID)
}

// TestRecoverToQueue_TimeoutReport tests that timed-out reports are marked as failed
func TestRecoverToQueue_TimeoutReport(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	// Create a report that started 25 hours ago (exceeds 24h timeout)
	startedAt := time.Now().Add(-25 * time.Hour)
	report := &model.Report{
		ID:         "test-report-timeout",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusAnalyzing,
		StartedAt:  &startedAt,
	}
	require.NoError(t, s.Report().Create(report))

	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: true}
	service := NewService(cfg, s, mockEnqueuer)

	service.RecoverToQueue(context.Background())

	// Should NOT enqueue the report
	assert.Empty(t, mockEnqueuer.enqueuedReports)

	// Should mark as failed
	recovered, err := s.Report().GetByID("test-report-timeout")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusFailed, recovered.Status)
	assert.Contains(t, recovered.ErrorMessage, "task timeout")
}

// TestRecoverToQueue_MaxRetryExceeded tests that reports with too many retries are marked as failed
func TestRecoverToQueue_MaxRetryExceeded(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	// Create a report with retry count >= max
	report := &model.Report{
		ID:         "test-report-retry",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
		RetryCount: 3,
	}
	require.NoError(t, s.Report().Create(report))

	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: true}
	service := NewService(cfg, s, mockEnqueuer)

	service.RecoverToQueue(context.Background())

	// Should NOT enqueue the report
	assert.Empty(t, mockEnqueuer.enqueuedReports)

	// Should mark as failed
	recovered, err := s.Report().GetByID("test-report-retry")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusFailed, recovered.Status)
	assert.Contains(t, recovered.ErrorMessage, "max retry count exceeded")
}

// TestRecoverToQueue_MultipleReports tests recovery of multiple reports with mixed conditions
func TestRecoverToQueue_MultipleReports(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	// Create multiple reports with different conditions
	startedAt := time.Now().Add(-1 * time.Hour)
	timeoutStartedAt := time.Now().Add(-25 * time.Hour)

	reports := []*model.Report{
		{
			ID:         "report-1",
			RepoURL:    "https://github.com/test/repo1",
			Ref:        "main",
			ReportType: "wiki",
			Status:     model.ReportStatusPending,
		},
		{
			ID:         "report-2",
			RepoURL:    "https://github.com/test/repo2",
			Ref:        "main",
			ReportType: "wiki",
			Status:     model.ReportStatusAnalyzing,
			StartedAt:  &startedAt,
		},
		{
			ID:         "report-3-timeout",
			RepoURL:    "https://github.com/test/repo3",
			Ref:        "main",
			ReportType: "wiki",
			Status:     model.ReportStatusGenerating,
			StartedAt:  &timeoutStartedAt,
		},
		{
			ID:         "report-4-retry",
			RepoURL:    "https://github.com/test/repo4",
			Ref:        "main",
			ReportType: "wiki",
			Status:     model.ReportStatusPending,
			RetryCount: 5,
		},
		{
			ID:         "report-5-completed",
			RepoURL:    "https://github.com/test/repo5",
			Ref:        "main",
			ReportType: "wiki",
			Status:     model.ReportStatusCompleted,
		},
	}

	for _, r := range reports {
		require.NoError(t, s.Report().Create(r))
	}

	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: true}
	service := NewService(cfg, s, mockEnqueuer)

	service.RecoverToQueue(context.Background())

	// Should enqueue 2 reports (report-1 and report-2)
	assert.Len(t, mockEnqueuer.enqueuedReports, 2)

	// Check that timeout and retry reports are marked as failed
	report3, err := s.Report().GetByID("report-3-timeout")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusFailed, report3.Status)

	report4, err := s.Report().GetByID("report-4-retry")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusFailed, report4.Status)

	// Completed report should remain completed
	report5, err := s.Report().GetByID("report-5-completed")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusCompleted, report5.Status)
}

// TestRecoverToQueue_EnqueueFailure tests handling when enqueue fails
func TestRecoverToQueue_EnqueueFailure(t *testing.T) {
	s, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	report := &model.Report{
		ID:         "test-report-enqueue-fail",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}
	require.NoError(t, s.Report().Create(report))

	// Mock enqueuer that fails
	mockEnqueuer := &mockTaskEnqueuer{enqueueReturn: false}
	service := NewService(cfg, s, mockEnqueuer)

	service.RecoverToQueue(context.Background())

	// Should mark as failed
	recovered, err := s.Report().GetByID("test-report-enqueue-fail")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusFailed, recovered.Status)
	assert.Contains(t, recovered.ErrorMessage, "could not enqueue task")
}

// TestShouldRecover tests the shouldRecover logic
func TestShouldRecover(t *testing.T) {
	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	service := &Service{cfg: cfg}

	tests := []struct {
		name          string
		report        *model.Report
		shouldRecover bool
		reasonSubstr  string
	}{
		{
			name: "pending report with no retries",
			report: &model.Report{
				Status:     model.ReportStatusPending,
				RetryCount: 0,
			},
			shouldRecover: true,
		},
		{
			name: "analyzing report within timeout",
			report: &model.Report{
				Status:     model.ReportStatusAnalyzing,
				StartedAt:  timePtr(time.Now().Add(-1 * time.Hour)),
				RetryCount: 0,
			},
			shouldRecover: true,
		},
		{
			name: "report with max retries",
			report: &model.Report{
				Status:     model.ReportStatusPending,
				RetryCount: 3,
			},
			shouldRecover: false,
			reasonSubstr:  "max retry count exceeded",
		},
		{
			name: "report exceeding timeout",
			report: &model.Report{
				Status:     model.ReportStatusAnalyzing,
				StartedAt:  timePtr(time.Now().Add(-25 * time.Hour)),
				RetryCount: 0,
			},
			shouldRecover: false,
			reasonSubstr:  "task timeout",
		},
		{
			name: "report with both timeout and retry issues",
			report: &model.Report{
				Status:     model.ReportStatusGenerating,
				StartedAt:  timePtr(time.Now().Add(-30 * time.Hour)),
				RetryCount: 5,
			},
			shouldRecover: false,
			reasonSubstr:  "max retry count exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRecover, reason := service.shouldRecover(tt.report)
			assert.Equal(t, tt.shouldRecover, shouldRecover)
			if !tt.shouldRecover {
				assert.Contains(t, reason, tt.reasonSubstr)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
