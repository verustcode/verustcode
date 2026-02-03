// Package notification provides notification services for task failure alerts.
// This file contains unit tests for email notification functionality.
package notification

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
)

// TestNewEmailNotifier tests creating a new email notifier
func TestNewEmailNotifier(t *testing.T) {
	cfg := &config.EmailNotificationConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "test@example.com",
		To:       []string{"recipient@example.com"},
	}

	notifier := NewEmailNotifier(cfg)
	require.NotNil(t, notifier)
	assert.Equal(t, "email", notifier.Name())
	assert.Equal(t, cfg, notifier.config)
}

// TestEmailNotifier_Name tests the Name method
func TestEmailNotifier_Name(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})
	assert.Equal(t, "email", notifier.Name())
}

// TestEmailNotifier_Send_ValidationErrors tests Send with various validation errors
func TestEmailNotifier_Send_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	tests := []struct {
		name string
		cfg  *config.EmailNotificationConfig
		want string
	}{
		{
			name: "missing SMTP host",
			cfg: &config.EmailNotificationConfig{
				SMTPPort: 587,
				From:     "test@example.com",
				To:       []string{"recipient@example.com"},
			},
			want: "SMTP host is not configured",
		},
		{
			name: "no recipients",
			cfg: &config.EmailNotificationConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				From:     "test@example.com",
				To:       []string{},
			},
			want: "no recipient email addresses configured",
		},
		{
			name: "missing sender",
			cfg: &config.EmailNotificationConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				To:       []string{"recipient@example.com"},
			},
			want: "sender email address is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewEmailNotifier(tt.cfg)
			err := notifier.Send(ctx, event)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// TestEmailNotifier_BuildSubject tests buildSubject method
func TestEmailNotifier_BuildSubject(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})

	tests := []struct {
		name  string
		event *Event
		want  string
	}{
		{
			name: "review completed",
			event: &Event{
				Type:     EventReviewCompleted,
				TaskID:   "review-001",
				TaskType: "review",
			},
			want: "[VerustCode] Review Task Completed: review-001",
		},
		{
			name: "review failed",
			event: &Event{
				Type:     EventReviewFailed,
				TaskID:   "review-002",
				TaskType: "review",
			},
			want: "[VerustCode] Review Task Failed: review-002",
		},
		{
			name: "report completed",
			event: &Event{
				Type:     EventReportCompleted,
				TaskID:   "report-001",
				TaskType: "report",
			},
			want: "[VerustCode] Report Task Completed: report-001",
		},
		{
			name: "report failed",
			event: &Event{
				Type:     EventReportFailed,
				TaskID:   "report-002",
				TaskType: "report",
			},
			want: "[VerustCode] Report Task Failed: report-002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := notifier.buildSubject(tt.event)
			assert.Equal(t, tt.want, subject)
		})
	}
}

// TestEmailNotifier_BuildBody tests buildBody method
func TestEmailNotifier_BuildBody(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})

	t.Run("review completed with duration", func(t *testing.T) {
		event := &Event{
			Type:      EventReviewCompleted,
			TaskID:    "review-001",
			TaskType:  "review",
			RepoURL:   "https://github.com/test/repo",
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Extra: map[string]interface{}{
				"duration_ms": int64(5000),
			},
		}

		body := notifier.buildBody(event)
		assert.Contains(t, body, "Review Task Completed")
		assert.Contains(t, body, "review-001")
		assert.Contains(t, body, "https://github.com/test/repo")
		assert.Contains(t, body, "2024-01-01 12:00:00")
		assert.Contains(t, body, "5.00 seconds")
		assert.Contains(t, body, "Sent by VerustCode Notification System")
	})

	t.Run("review failed with error", func(t *testing.T) {
		event := &Event{
			Type:         EventReviewFailed,
			TaskID:       "review-002",
			TaskType:     "review",
			RepoURL:      "https://github.com/test/repo",
			ErrorMessage: "Test error message",
			Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		}

		body := notifier.buildBody(event)
		assert.Contains(t, body, "Review Task Failed")
		assert.Contains(t, body, "review-002")
		assert.Contains(t, body, "Error:")
		assert.Contains(t, body, "Test error message")
		assert.NotContains(t, body, "Duration:")
	})

	t.Run("report completed with extra info", func(t *testing.T) {
		event := &Event{
			Type:      EventReportCompleted,
			TaskID:    "report-001",
			TaskType:  "report",
			RepoURL:   "https://github.com/test/repo",
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Extra: map[string]interface{}{
				"duration_ms":   int64(10000),
				"section_count": 5,
			},
		}

		body := notifier.buildBody(event)
		assert.Contains(t, body, "Report Task Completed")
		assert.Contains(t, body, "10.00 seconds")
		assert.Contains(t, body, "Additional Information:")
		assert.Contains(t, body, "section_count: 5")
		// duration_ms should not appear in Additional Information
		assert.NotContains(t, strings.Split(body, "Additional Information:")[1], "duration_ms")
	})
}

// TestEmailNotifier_BuildMessage tests buildMessage method
func TestEmailNotifier_BuildMessage(t *testing.T) {
	cfg := &config.EmailNotificationConfig{
		From: "sender@example.com",
		To:   []string{"recipient1@example.com", "recipient2@example.com"},
	}
	notifier := NewEmailNotifier(cfg)

	subject := "Test Subject"
	body := "Test body content"

	msg := notifier.buildMessage(subject, body)

	assert.Contains(t, msg, "From: sender@example.com")
	assert.Contains(t, msg, "To: recipient1@example.com, recipient2@example.com")
	assert.Contains(t, msg, "Subject: Test Subject")
	assert.Contains(t, msg, "MIME-Version: 1.0")
	assert.Contains(t, msg, "Content-Type: text/plain; charset=UTF-8")
	assert.Contains(t, msg, "Test body content")

	// Check CRLF line endings
	assert.Contains(t, msg, "\r\n")
}

// TestEmailNotifier_BuildMessage_SingleRecipient tests buildMessage with single recipient
func TestEmailNotifier_BuildMessage_SingleRecipient(t *testing.T) {
	cfg := &config.EmailNotificationConfig{
		From: "sender@example.com",
		To:   []string{"recipient@example.com"},
	}
	notifier := NewEmailNotifier(cfg)

	msg := notifier.buildMessage("Subject", "Body")
	assert.Contains(t, msg, "To: recipient@example.com")
}

// TestEmailNotifier_Send_WithAuth tests Send with authentication configured
func TestEmailNotifier_Send_WithAuth(t *testing.T) {
	cfg := &config.EmailNotificationConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "test@example.com",
		To:       []string{"recipient@example.com"},
		Username: "user",
		Password: "pass",
	}

	notifier := NewEmailNotifier(cfg)
	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	// This will fail because we don't have a real SMTP server, but we can test the validation
	ctx := context.Background()
	err := notifier.Send(ctx, event)
	// Should fail with connection error, not validation error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

// TestEmailNotifier_Send_WithoutAuth tests Send without authentication
func TestEmailNotifier_Send_WithoutAuth(t *testing.T) {
	cfg := &config.EmailNotificationConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "test@example.com",
		To:       []string{"recipient@example.com"},
		// No username/password
	}

	notifier := NewEmailNotifier(cfg)
	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	err := notifier.Send(ctx, event)
	// Should fail with connection error, not validation error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

// TestEmailNotifier_Send_Port465 tests Send with port 465 (TLS)
func TestEmailNotifier_Send_Port465(t *testing.T) {
	cfg := &config.EmailNotificationConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 465, // TLS port
		From:     "test@example.com",
		To:       []string{"recipient@example.com"},
	}

	notifier := NewEmailNotifier(cfg)
	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	err := notifier.Send(ctx, event)
	// Should fail with connection error (TLS connection), not validation error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

// TestEmailNotifier_BuildBody_EmptyExtra tests buildBody with empty extra map
func TestEmailNotifier_BuildBody_EmptyExtra(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
		Extra:     make(map[string]interface{}),
	}

	body := notifier.buildBody(event)
	assert.Contains(t, body, "Review Task Completed")
	assert.NotContains(t, body, "Additional Information:")
}

// TestEmailNotifier_BuildBody_MultipleRecipients tests buildBody formatting
func TestEmailNotifier_BuildBody_MultipleRecipients(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Extra: map[string]interface{}{
			"duration_ms": int64(3000),
			"findings":    10,
		},
	}

	body := notifier.buildBody(event)

	// Check structure
	assert.Contains(t, body, "Review Task Completed")
	assert.Contains(t, body, "================================\n")
	assert.Contains(t, body, "Task ID: review-001")
	assert.Contains(t, body, "Repository: https://github.com/test/repo")
	assert.Contains(t, body, "Duration: 3.00 seconds")
	assert.Contains(t, body, "findings: 10")
}

// TestEmailNotifier_BuildMessage_Headers tests message header formatting
func TestEmailNotifier_BuildMessage_Headers(t *testing.T) {
	cfg := &config.EmailNotificationConfig{
		From: "sender@example.com",
		To:   []string{"recipient@example.com"},
	}
	notifier := NewEmailNotifier(cfg)

	msg := notifier.buildMessage("Test Subject", "Test Body")

	// Check header order and format
	lines := strings.Split(msg, "\r\n")
	assert.Equal(t, "From: sender@example.com", lines[0])
	assert.Equal(t, "To: recipient@example.com", lines[1])
	assert.Equal(t, "Subject: Test Subject", lines[2])
	assert.Equal(t, "MIME-Version: 1.0", lines[3])
	assert.Equal(t, "Content-Type: text/plain; charset=UTF-8", lines[4])
	assert.Equal(t, "", lines[5]) // Empty line before body
	assert.Equal(t, "Test Body", lines[6])
}

// TestEmailNotifier_BuildBody_ErrorFormatting tests error message formatting
func TestEmailNotifier_BuildBody_ErrorFormatting(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})

	event := &Event{
		Type:         EventReviewFailed,
		TaskID:       "review-001",
		TaskType:     "review",
		RepoURL:      "https://github.com/test/repo",
		ErrorMessage: "Multi-line\nerror\nmessage",
		Timestamp:    time.Now(),
	}

	body := notifier.buildBody(event)
	assert.Contains(t, body, "Error:")
	assert.Contains(t, body, "Multi-line\nerror\nmessage")
}

// TestEmailNotifier_BuildBody_TimeFormat tests timestamp formatting
func TestEmailNotifier_BuildBody_TimeFormat(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})

	// Test with UTC time
	utcTime := time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC)
	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: utcTime,
	}

	body := notifier.buildBody(event)
	// Should contain formatted time
	assert.Contains(t, body, "2024-01-01 12:30:45 UTC")
}

// TestEmailNotifier_BuildBody_DurationFormatting tests duration formatting
func TestEmailNotifier_BuildBody_DurationFormatting(t *testing.T) {
	notifier := NewEmailNotifier(&config.EmailNotificationConfig{})

	tests := []struct {
		name     string
		duration int64
		want     string
	}{
		{
			name:     "seconds",
			duration: 1000,
			want:     "1.00 seconds",
		},
		{
			name:     "milliseconds",
			duration: 500,
			want:     "0.50 seconds",
		},
		{
			name:     "minutes",
			duration: 120000,
			want:     "120.00 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				Type:      EventReviewCompleted,
				TaskID:    "review-001",
				TaskType:  "review",
				RepoURL:   "https://github.com/test/repo",
				Timestamp: time.Now(),
				Extra: map[string]interface{}{
					"duration_ms": tt.duration,
				},
			}

			body := notifier.buildBody(event)
			assert.Contains(t, body, tt.want)
		})
	}
}
