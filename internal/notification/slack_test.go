// Package notification provides notification services for task failure alerts.
// This file contains unit tests for Slack notification functionality.
package notification

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
)

// TestNewSlackNotifier tests creating a new Slack notifier
func TestNewSlackNotifier(t *testing.T) {
	cfg := &config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
		Channel:    "#test-channel",
	}

	notifier := NewSlackNotifier(cfg)
	require.NotNil(t, notifier)
	assert.Equal(t, "slack", notifier.Name())
	assert.Equal(t, cfg, notifier.config)
	assert.NotNil(t, notifier.client)
}

// TestSlackNotifier_Name tests the Name method
func TestSlackNotifier_Name(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{})
	assert.Equal(t, "slack", notifier.Name())
}

// TestSlackNotifier_Send_ValidationError tests Send with missing webhook URL
func TestSlackNotifier_Send_ValidationError(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		// WebhookURL is empty
	})

	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	err := notifier.Send(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Slack webhook URL is not configured")
}

// TestSlackNotifier_BuildMessage tests buildMessage method
func TestSlackNotifier_BuildMessage(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
		Channel:    "#test-channel",
	})

	tests := []struct {
		name  string
		event *Event
		check func(*testing.T, *SlackMessage)
	}{
		{
			name: "review completed",
			event: &Event{
				Type:      EventReviewCompleted,
				TaskID:    "review-001",
				TaskType:  "review",
				RepoURL:   "https://github.com/test/repo",
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Extra: map[string]interface{}{
					"duration_ms": int64(5000),
				},
			},
			check: func(t *testing.T, msg *SlackMessage) {
				assert.Contains(t, msg.Text, ":white_check_mark:")
				assert.Contains(t, msg.Text, "Review Task Completed")
				assert.Equal(t, "#test-channel", msg.Channel)
				assert.Len(t, msg.Attachments, 1)
				attachment := msg.Attachments[0]
				assert.Equal(t, "good", attachment.Color)
				assert.Contains(t, attachment.Title, "Review Task")
				assert.Contains(t, attachment.Footer, "VerustCode")
				// Check fields
				foundRepo := false
				foundDuration := false
				for _, field := range attachment.Fields {
					if field.Title == "Repository" {
						assert.Equal(t, "https://github.com/test/repo", field.Value)
						foundRepo = true
					}
					if field.Title == "Duration" {
						assert.Contains(t, field.Value, "5.00")
						foundDuration = true
					}
				}
				assert.True(t, foundRepo, "Repository field should be present")
				assert.True(t, foundDuration, "Duration field should be present")
			},
		},
		{
			name: "review failed",
			event: &Event{
				Type:         EventReviewFailed,
				TaskID:       "review-002",
				TaskType:     "review",
				RepoURL:      "https://github.com/test/repo",
				ErrorMessage: "Test error message",
				Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			check: func(t *testing.T, msg *SlackMessage) {
				assert.Contains(t, msg.Text, ":x:")
				assert.Contains(t, msg.Text, "Review Task Failed")
				assert.Len(t, msg.Attachments, 1)
				attachment := msg.Attachments[0]
				assert.Equal(t, "danger", attachment.Color)
				// Check error field
				foundError := false
				for _, field := range attachment.Fields {
					if field.Title == "Error" {
						assert.Contains(t, field.Value, "Test error message")
						foundError = true
					}
				}
				assert.True(t, foundError, "Error field should be present")
			},
		},
		{
			name: "report completed",
			event: &Event{
				Type:      EventReportCompleted,
				TaskID:    "report-001",
				TaskType:  "report",
				RepoURL:   "https://github.com/test/repo",
				Timestamp: time.Now(),
			},
			check: func(t *testing.T, msg *SlackMessage) {
				assert.Contains(t, msg.Text, "Report Task Completed")
			},
		},
		{
			name: "report failed",
			event: &Event{
				Type:         EventReportFailed,
				TaskID:       "report-002",
				TaskType:     "report",
				RepoURL:      "https://github.com/test/repo",
				ErrorMessage: "Report error",
				Timestamp:    time.Now(),
			},
			check: func(t *testing.T, msg *SlackMessage) {
				assert.Contains(t, msg.Text, "Report Task Failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := notifier.buildMessage(tt.event)
			require.NotNil(t, msg)
			tt.check(t, msg)
		})
	}
}

// TestSlackNotifier_BuildMessage_NoChannel tests buildMessage without channel
func TestSlackNotifier_BuildMessage_NoChannel(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
		// No channel specified
	})

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	msg := notifier.buildMessage(event)
	assert.Empty(t, msg.Channel, "Channel should be empty when not configured")
}

// TestSlackNotifier_TruncateText tests truncateText method
func TestSlackNotifier_TruncateText(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{})

	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{
			name:   "short text",
			text:   "Short",
			maxLen: 100,
			want:   "Short",
		},
		{
			name:   "exact length",
			text:   strings.Repeat("a", 100),
			maxLen: 100,
			want:   strings.Repeat("a", 100),
		},
		{
			name:   "long text",
			text:   strings.Repeat("a", 200),
			maxLen: 100,
			want:   strings.Repeat("a", 97) + "...",
		},
		{
			name:   "very long text",
			text:   strings.Repeat("b", 1000),
			maxLen: 50,
			want:   strings.Repeat("b", 47) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := notifier.truncateText(tt.text, tt.maxLen)
			assert.Equal(t, tt.want, result)
			assert.LessOrEqual(t, len(result), tt.maxLen)
		})
	}
}

// TestSlackNotifier_BuildMessage_Fields tests message fields structure
func TestSlackNotifier_BuildMessage_Fields(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
	})

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Extra: map[string]interface{}{
			"duration_ms": int64(3000),
		},
	}

	msg := notifier.buildMessage(event)
	require.Len(t, msg.Attachments, 1)
	attachment := msg.Attachments[0]

	// Should have Repository, Task ID, Time, and Duration fields
	assert.GreaterOrEqual(t, len(attachment.Fields), 3)

	fieldMap := make(map[string]SlackField)
	for _, field := range attachment.Fields {
		fieldMap[field.Title] = field
	}

	assert.Contains(t, fieldMap, "Repository")
	assert.Equal(t, "https://github.com/test/repo", fieldMap["Repository"].Value)
	assert.False(t, fieldMap["Repository"].Short)

	assert.Contains(t, fieldMap, "Task ID")
	assert.Equal(t, "review-001", fieldMap["Task ID"].Value)
	assert.True(t, fieldMap["Task ID"].Short)

	assert.Contains(t, fieldMap, "Time")
	assert.Contains(t, fieldMap["Time"].Value, "2024-01-01")
	assert.True(t, fieldMap["Time"].Short)

	assert.Contains(t, fieldMap, "Duration")
	assert.Contains(t, fieldMap["Duration"].Value, "3.00")
	assert.True(t, fieldMap["Duration"].Short)
}

// TestSlackNotifier_BuildMessage_Timestamp tests timestamp in attachment
func TestSlackNotifier_BuildMessage_Timestamp(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
	})

	eventTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: eventTime,
	}

	msg := notifier.buildMessage(event)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, eventTime.Unix(), msg.Attachments[0].Timestamp)
}

// TestSlackNotifier_BuildMessage_ErrorTruncation tests error message truncation
func TestSlackNotifier_BuildMessage_ErrorTruncation(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
	})

	longError := strings.Repeat("error ", 200) // Very long error message
	event := &Event{
		Type:         EventReviewFailed,
		TaskID:       "review-001",
		TaskType:     "review",
		RepoURL:      "https://github.com/test/repo",
		ErrorMessage: longError,
		Timestamp:    time.Now(),
	}

	msg := notifier.buildMessage(event)
	require.Len(t, msg.Attachments, 1)
	attachment := msg.Attachments[0]

	var errorField *SlackField
	for i := range attachment.Fields {
		if attachment.Fields[i].Title == "Error" {
			errorField = &attachment.Fields[i]
			break
		}
	}

	require.NotNil(t, errorField)
	assert.LessOrEqual(t, len(errorField.Value), 500)
	assert.Contains(t, errorField.Value, "...")
}

// TestSlackNotifier_Send_MarshalError tests Send with JSON marshal error
func TestSlackNotifier_Send_MarshalError(t *testing.T) {
	// This test is hard to trigger without mocking, but we can test the structure
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
	})

	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	// Verify message can be marshaled
	msg := notifier.buildMessage(event)
	_, err := json.Marshal(msg)
	assert.NoError(t, err, "Message should be JSON marshalable")
}

// TestSlackNotifier_BuildMessage_NoDuration tests buildMessage without duration
func TestSlackNotifier_BuildMessage_NoDuration(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
	})

	event := &Event{
		Type:      EventReviewFailed, // Failed event, no duration
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	msg := notifier.buildMessage(event)
	require.Len(t, msg.Attachments, 1)
	attachment := msg.Attachments[0]

	// Should not have Duration field for failed events
	for _, field := range attachment.Fields {
		assert.NotEqual(t, "Duration", field.Title, "Failed events should not have Duration field")
	}
}

// TestSlackNotifier_BuildMessage_MultipleFields tests multiple field types
func TestSlackNotifier_BuildMessage_MultipleFields(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
	})

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

	msg := notifier.buildMessage(event)
	require.Len(t, msg.Attachments, 1)
	attachment := msg.Attachments[0]

	// Verify field structure
	assert.GreaterOrEqual(t, len(attachment.Fields), 4, "Should have at least 4 fields")

	// Check that short fields are properly marked
	shortCount := 0
	longCount := 0
	for _, field := range attachment.Fields {
		if field.Short {
			shortCount++
		} else {
			longCount++
		}
	}

	assert.Greater(t, shortCount, 0, "Should have some short fields")
	assert.Greater(t, longCount, 0, "Should have some long fields")
}

// TestSlackMessage_JSONStructure tests JSON structure of SlackMessage
func TestSlackMessage_JSONStructure(t *testing.T) {
	notifier := NewSlackNotifier(&config.SlackNotificationConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
		Channel:    "#test",
	})

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	msg := notifier.buildMessage(event)
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Verify JSON structure
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Contains(t, decoded, "text")
	assert.Contains(t, decoded, "attachments")
	assert.Contains(t, decoded, "channel")
	assert.Equal(t, "#test", decoded["channel"])

	attachments, ok := decoded["attachments"].([]interface{})
	require.True(t, ok)
	require.Len(t, attachments, 1)

	attachment, ok := attachments[0].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, attachment, "color")
	assert.Contains(t, attachment, "title")
	assert.Contains(t, attachment, "fields")
	assert.Contains(t, attachment, "footer")
	assert.Contains(t, attachment, "ts")
}
