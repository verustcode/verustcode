package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/pkg/logger"
)

// FeishuNotifier sends notifications via Feishu/Lark webhook
type FeishuNotifier struct {
	config *config.FeishuNotificationConfig
	client *http.Client
}

// FeishuMessage represents a Feishu message payload
type FeishuMessage struct {
	Timestamp string      `json:"timestamp,omitempty"`
	Sign      string      `json:"sign,omitempty"`
	MsgType   string      `json:"msg_type"`
	Card      *FeishuCard `json:"card,omitempty"`
}

// FeishuCard represents an interactive card message
type FeishuCard struct {
	Config   FeishuCardConfig    `json:"config"`
	Header   FeishuCardHeader    `json:"header"`
	Elements []FeishuCardElement `json:"elements"`
}

// FeishuCardConfig represents card configuration
type FeishuCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

// FeishuCardHeader represents card header
type FeishuCardHeader struct {
	Title    FeishuCardText `json:"title"`
	Template string         `json:"template"`
}

// FeishuCardText represents text content
type FeishuCardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// FeishuCardElement represents a card element
type FeishuCardElement struct {
	Tag      string              `json:"tag"`
	Text     *FeishuCardText     `json:"text,omitempty"`
	Fields   []FeishuField       `json:"fields,omitempty"`
	Content  string              `json:"content,omitempty"`
	Elements []FeishuNoteElement `json:"elements,omitempty"` // Used for note tag
}

// FeishuNoteElement represents an element in note tag
type FeishuNoteElement struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// FeishuField represents a field in card
type FeishuField struct {
	IsShort bool           `json:"is_short"`
	Text    FeishuCardText `json:"text"`
}

// NewFeishuNotifier creates a new Feishu notifier
func NewFeishuNotifier(cfg *config.FeishuNotificationConfig) *FeishuNotifier {
	return &FeishuNotifier{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the notifier name
func (f *FeishuNotifier) Name() string {
	return "feishu"
}

// Send sends a notification to Feishu
func (f *FeishuNotifier) Send(ctx context.Context, event *Event) error {
	if f.config.WebhookURL == "" {
		return fmt.Errorf("Feishu webhook URL is not configured")
	}

	// Build Feishu message
	msg := f.buildMessage(event)

	// Add signature if secret is configured
	if f.config.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		sign := f.computeSignature(timestamp)
		msg.Timestamp = timestamp
		msg.Sign = sign
	}

	// Marshal to JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Feishu message: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Feishu request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	logger.Debug("Sending Feishu notification",
		zap.String("event_type", string(event.Type)),
	)

	// Send request
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Feishu request: %w", err)
	}
	defer resp.Body.Close()

	// Read and parse response
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	// Parse response to check for errors
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if result.Code != 0 {
			return fmt.Errorf("Feishu returned error: code=%d, msg=%s", result.Code, result.Msg)
		}
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Feishu returned error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	logger.Debug("Feishu notification sent successfully")

	return nil
}

// buildMessage builds a Feishu interactive card message
func (f *FeishuNotifier) buildMessage(event *Event) *FeishuMessage {
	taskType := "Review"
	if event.TaskType == "report" {
		taskType = "Report"
	}

	// Determine if this is a success or failure event
	isSuccess := event.Type == EventReviewCompleted || event.Type == EventReportCompleted
	var headerTemplate, emoji, statusText string
	if isSuccess {
		headerTemplate = "green" // Green for success
		emoji = "✅"
		statusText = "完成"
	} else {
		headerTemplate = "red" // Red for failures
		emoji = "⚠️"
		statusText = "失败"
	}

	// Build elements
	elements := []FeishuCardElement{
		{
			Tag: "div",
			Fields: []FeishuField{
				{
					IsShort: true,
					Text: FeishuCardText{
						Tag:     "lark_md",
						Content: fmt.Sprintf("**任务 ID**\n%s", event.TaskID),
					},
				},
				{
					IsShort: true,
					Text: FeishuCardText{
						Tag:     "lark_md",
						Content: fmt.Sprintf("**时间**\n%s", event.Timestamp.Format("2006-01-02 15:04:05")),
					},
				},
			},
		},
		{
			Tag: "div",
			Text: &FeishuCardText{
				Tag:     "lark_md",
				Content: fmt.Sprintf("**仓库**\n%s", event.RepoURL),
			},
		},
	}

	// Add error message for failure events
	if !isSuccess && event.ErrorMessage != "" {
		elements = append(elements,
			FeishuCardElement{Tag: "hr"},
			FeishuCardElement{
				Tag: "div",
				Text: &FeishuCardText{
					Tag:     "lark_md",
					Content: fmt.Sprintf("**错误信息**\n```\n%s\n```", f.truncateText(event.ErrorMessage, 500)),
				},
			},
		)
	}

	// Add duration for success events
	if isSuccess {
		if duration, ok := event.Extra["duration_ms"].(int64); ok {
			elements = append(elements,
				FeishuCardElement{
					Tag: "div",
					Text: &FeishuCardText{
						Tag:     "lark_md",
						Content: fmt.Sprintf("**耗时**\n%.2f 秒", float64(duration)/1000),
					},
				},
			)
		}
	}

	// Add footer
	elements = append(elements, FeishuCardElement{
		Tag: "note",
		Elements: []FeishuNoteElement{
			{
				Tag:     "plain_text",
				Content: "来自 VerustCode 通知系统",
			},
		},
	})

	card := &FeishuCard{
		Config: FeishuCardConfig{
			WideScreenMode: true,
		},
		Header: FeishuCardHeader{
			Title: FeishuCardText{
				Tag:     "plain_text",
				Content: fmt.Sprintf("%s %s 任务%s", emoji, taskType, statusText),
			},
			Template: headerTemplate,
		},
		Elements: elements,
	}

	return &FeishuMessage{
		MsgType: "interactive",
		Card:    card,
	}
}

// computeSignature computes HMAC-SHA256 signature for Feishu
func (f *FeishuNotifier) computeSignature(timestamp string) string {
	stringToSign := timestamp + "\n" + f.config.Secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// truncateText truncates text to a maximum length
func (f *FeishuNotifier) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
