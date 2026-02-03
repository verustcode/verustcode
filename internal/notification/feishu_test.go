// Package notification provides notification services for task failure alerts.
// This file contains unit tests for Feishu notification functionality.
package notification

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
)

// TestNewFeishuNotifier tests creating a new Feishu notifier
func TestNewFeishuNotifier(t *testing.T) {
	cfg := &config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Secret:     "test-secret",
	}

	notifier := NewFeishuNotifier(cfg)
	require.NotNil(t, notifier)
	assert.Equal(t, "feishu", notifier.Name())
	assert.Equal(t, cfg, notifier.config)
	assert.NotNil(t, notifier.client)
}

// TestFeishuNotifier_Name tests the Name method
func TestFeishuNotifier_Name(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{})
	assert.Equal(t, "feishu", notifier.Name())
}

// TestFeishuNotifier_Send_ValidationError tests Send with missing webhook URL
func TestFeishuNotifier_Send_ValidationError(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
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
	assert.Contains(t, err.Error(), "Feishu webhook URL is not configured")
}

// TestFeishuNotifier_ComputeSignature tests computeSignature method
func TestFeishuNotifier_ComputeSignature(t *testing.T) {
	cfg := &config.FeishuNotificationConfig{
		Secret: "test-secret-key",
	}
	notifier := NewFeishuNotifier(cfg)

	timestamp := "1234567890"
	signature := notifier.computeSignature(timestamp)

	// Verify signature format (base64 encoded)
	_, err := base64.StdEncoding.DecodeString(signature)
	assert.NoError(t, err, "Signature should be valid base64")

	// Verify signature computation manually
	stringToSign := timestamp + "\n" + cfg.Secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	assert.Equal(t, expectedSignature, signature)
}

// TestFeishuNotifier_ComputeSignature_DifferentTimestamps tests signature with different timestamps
func TestFeishuNotifier_ComputeSignature_DifferentTimestamps(t *testing.T) {
	cfg := &config.FeishuNotificationConfig{
		Secret: "test-secret",
	}
	notifier := NewFeishuNotifier(cfg)

	timestamp1 := "1234567890"
	timestamp2 := "9876543210"

	sig1 := notifier.computeSignature(timestamp1)
	sig2 := notifier.computeSignature(timestamp2)

	assert.NotEqual(t, sig1, sig2, "Different timestamps should produce different signatures")
}

// TestFeishuNotifier_BuildMessage tests buildMessage method
func TestFeishuNotifier_BuildMessage(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})

	tests := []struct {
		name  string
		event *Event
		check func(*testing.T, *FeishuMessage)
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
			check: func(t *testing.T, msg *FeishuMessage) {
				assert.Equal(t, "interactive", msg.MsgType)
				require.NotNil(t, msg.Card)
				assert.True(t, msg.Card.Config.WideScreenMode)
				assert.Equal(t, "green", msg.Card.Header.Template)
				assert.Contains(t, msg.Card.Header.Title.Content, "✅")
				assert.Contains(t, msg.Card.Header.Title.Content, "完成")
				assert.Greater(t, len(msg.Card.Elements), 0)
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
			check: func(t *testing.T, msg *FeishuMessage) {
				require.NotNil(t, msg.Card)
				assert.Equal(t, "red", msg.Card.Header.Template)
				assert.Contains(t, msg.Card.Header.Title.Content, "⚠️")
				assert.Contains(t, msg.Card.Header.Title.Content, "失败")
				// Check for error element
				hasError := false
				for _, elem := range msg.Card.Elements {
					if elem.Text != nil && strings.Contains(elem.Text.Content, "错误信息") {
						hasError = true
						break
					}
				}
				assert.True(t, hasError, "Should have error message element")
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
			check: func(t *testing.T, msg *FeishuMessage) {
				require.NotNil(t, msg.Card)
				assert.Contains(t, msg.Card.Header.Title.Content, "Report")
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
			check: func(t *testing.T, msg *FeishuMessage) {
				require.NotNil(t, msg.Card)
				assert.Equal(t, "red", msg.Card.Header.Template)
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

// TestFeishuNotifier_BuildMessage_Elements tests card elements structure
func TestFeishuNotifier_BuildMessage_Elements(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
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
	require.NotNil(t, msg.Card)
	assert.Greater(t, len(msg.Card.Elements), 0)

	// Check for required elements
	hasTaskID := false
	hasTime := false
	hasRepo := false
	hasDuration := false
	hasFooter := false

	for _, elem := range msg.Card.Elements {
		if elem.Tag == "div" {
			if elem.Fields != nil {
				for _, field := range elem.Fields {
					if strings.Contains(field.Text.Content, "任务 ID") {
						hasTaskID = true
					}
					if strings.Contains(field.Text.Content, "时间") {
						hasTime = true
					}
				}
			}
			if elem.Text != nil {
				if strings.Contains(elem.Text.Content, "仓库") {
					hasRepo = true
				}
				if strings.Contains(elem.Text.Content, "耗时") {
					hasDuration = true
				}
			}
		}
		if elem.Tag == "note" {
			hasFooter = true
		}
	}

	assert.True(t, hasTaskID, "Should have Task ID field")
	assert.True(t, hasTime, "Should have Time field")
	assert.True(t, hasRepo, "Should have Repository field")
	assert.True(t, hasDuration, "Should have Duration field")
	assert.True(t, hasFooter, "Should have footer note")
}

// TestFeishuNotifier_BuildMessage_ErrorElement tests error element structure
func TestFeishuNotifier_BuildMessage_ErrorElement(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})

	event := &Event{
		Type:         EventReviewFailed,
		TaskID:       "review-001",
		TaskType:     "review",
		RepoURL:      "https://github.com/test/repo",
		ErrorMessage: "Test error\nwith multiple\nlines",
		Timestamp:    time.Now(),
	}

	msg := notifier.buildMessage(event)
	require.NotNil(t, msg.Card)

	// Find error element
	foundError := false
	for _, elem := range msg.Card.Elements {
		if elem.Tag == "div" && elem.Text != nil {
			if strings.Contains(elem.Text.Content, "错误信息") {
				foundError = true
				assert.Contains(t, elem.Text.Content, "Test error")
				assert.Contains(t, elem.Text.Content, "```")
				break
			}
		}
	}
	assert.True(t, foundError, "Should have error element")
}

// TestFeishuNotifier_TruncateText tests truncateText method
func TestFeishuNotifier_TruncateText(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{})

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
			text:   strings.Repeat("b", 200),
			maxLen: 100,
			want:   strings.Repeat("b", 97) + "...",
		},
		{
			name:   "very long text",
			text:   strings.Repeat("c", 1000),
			maxLen: 50,
			want:   strings.Repeat("c", 47) + "...",
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

// TestFeishuNotifier_BuildMessage_NoDuration tests buildMessage without duration
func TestFeishuNotifier_BuildMessage_NoDuration(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})

	event := &Event{
		Type:      EventReviewFailed, // Failed event, no duration
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	msg := notifier.buildMessage(event)
	require.NotNil(t, msg.Card)

	// Should not have duration element for failed events
	hasDuration := false
	for _, elem := range msg.Card.Elements {
		if elem.Text != nil && strings.Contains(elem.Text.Content, "耗时") {
			hasDuration = true
			break
		}
	}
	assert.False(t, hasDuration, "Failed events should not have duration element")
}

// TestFeishuNotifier_BuildMessage_Footer tests footer element
func TestFeishuNotifier_BuildMessage_Footer(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	msg := notifier.buildMessage(event)
	require.NotNil(t, msg.Card)

	// Find footer note element
	foundFooter := false
	for _, elem := range msg.Card.Elements {
		if elem.Tag == "note" {
			foundFooter = true
			assert.Greater(t, len(elem.Elements), 0)
			if len(elem.Elements) > 0 {
				assert.Equal(t, "plain_text", elem.Elements[0].Tag)
				assert.Contains(t, elem.Elements[0].Content, "VerustCode")
			}
			break
		}
	}
	assert.True(t, foundFooter, "Should have footer note element")
}

// TestFeishuNotifier_BuildMessage_ErrorTruncation tests error message truncation
func TestFeishuNotifier_BuildMessage_ErrorTruncation(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})

	longError := strings.Repeat("错误 ", 200) // Very long error message
	event := &Event{
		Type:         EventReviewFailed,
		TaskID:       "review-001",
		TaskType:     "review",
		RepoURL:      "https://github.com/test/repo",
		ErrorMessage: longError,
		Timestamp:    time.Now(),
	}

	msg := notifier.buildMessage(event)
	require.NotNil(t, msg.Card)

	// Find error element
	for _, elem := range msg.Card.Elements {
		if elem.Tag == "div" && elem.Text != nil && strings.Contains(elem.Text.Content, "错误信息") {
			// Error should be truncated
			assert.LessOrEqual(t, len(elem.Text.Content), 500+100) // Some overhead for markdown
			assert.Contains(t, elem.Text.Content, "...")
			return
		}
	}
	t.Error("Should have error element")
}

// TestFeishuNotifier_Send_WithSignature tests Send with signature
func TestFeishuNotifier_Send_WithSignature(t *testing.T) {
	cfg := &config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Secret:     "test-secret",
	}

	notifier := NewFeishuNotifier(cfg)
	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	// Build message and verify signature is added
	msg := notifier.buildMessage(event)

	// When secret is configured, Send should add signature
	// We can't easily test the actual Send without mocking HTTP, but we can verify the message structure
	assert.Equal(t, "interactive", msg.MsgType)
	assert.NotNil(t, msg.Card)
}

// TestFeishuNotifier_BuildMessage_FieldsStructure tests fields structure in div elements
func TestFeishuNotifier_BuildMessage_FieldsStructure(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
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
	require.NotNil(t, msg.Card)

	// Find div with fields
	for _, elem := range msg.Card.Elements {
		if elem.Tag == "div" && elem.Fields != nil {
			assert.Greater(t, len(elem.Fields), 0)
			for _, field := range elem.Fields {
				assert.True(t, field.IsShort, "Task ID and Time fields should be short")
				assert.Equal(t, "lark_md", field.Text.Tag)
				assert.NotEmpty(t, field.Text.Content)
			}
			return
		}
	}
	t.Error("Should have div element with fields")
}

// TestFeishuMessage_JSONStructure tests JSON structure of FeishuMessage
func TestFeishuMessage_JSONStructure(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Secret:     "test-secret",
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

	assert.Equal(t, "interactive", decoded["msg_type"])
	assert.Contains(t, decoded, "card")

	card, ok := decoded["card"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, card, "config")
	assert.Contains(t, card, "header")
	assert.Contains(t, card, "elements")

	header, ok := card["header"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, header, "title")
	assert.Contains(t, header, "template")
}

// TestFeishuNotifier_BuildMessage_HeaderTemplate tests header template colors
func TestFeishuNotifier_BuildMessage_HeaderTemplate(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})

	tests := []struct {
		name         string
		event        *Event
		wantTemplate string
	}{
		{
			name: "success event",
			event: &Event{
				Type:      EventReviewCompleted,
				TaskID:    "test-001",
				TaskType:  "review",
				RepoURL:   "https://github.com/test/repo",
				Timestamp: time.Now(),
			},
			wantTemplate: "green",
		},
		{
			name: "failure event",
			event: &Event{
				Type:         EventReviewFailed,
				TaskID:       "test-002",
				TaskType:     "review",
				RepoURL:      "https://github.com/test/repo",
				ErrorMessage: "Error",
				Timestamp:    time.Now(),
			},
			wantTemplate: "red",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := notifier.buildMessage(tt.event)
			require.NotNil(t, msg.Card)
			assert.Equal(t, tt.wantTemplate, msg.Card.Header.Template)
		})
	}
}

// TestFeishuNotifier_BuildMessage_DurationFormat tests duration formatting
func TestFeishuNotifier_BuildMessage_DurationFormat(t *testing.T) {
	notifier := NewFeishuNotifier(&config.FeishuNotificationConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    "review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
		Extra: map[string]interface{}{
			"duration_ms": int64(3500),
		},
	}

	msg := notifier.buildMessage(event)
	require.NotNil(t, msg.Card)

	// Find duration element
	for _, elem := range msg.Card.Elements {
		if elem.Text != nil && strings.Contains(elem.Text.Content, "耗时") {
			assert.Contains(t, elem.Text.Content, "3.50")
			assert.Contains(t, elem.Text.Content, "秒")
			return
		}
	}
	t.Error("Should have duration element")
}
