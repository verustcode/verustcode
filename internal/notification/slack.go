package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/pkg/logger"
)

// SlackNotifier sends notifications via Slack incoming webhook
type SlackNotifier struct {
	config *config.SlackNotificationConfig
	client *http.Client
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color      string       `json:"color"`
	Title      string       `json:"title"`
	Text       string       `json:"text"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// SlackField represents a field in Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(cfg *config.SlackNotificationConfig) *SlackNotifier {
	return &SlackNotifier{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the notifier name
func (s *SlackNotifier) Name() string {
	return "slack"
}

// Send sends a notification to Slack
func (s *SlackNotifier) Send(ctx context.Context, event *Event) error {
	if s.config.WebhookURL == "" {
		return fmt.Errorf("Slack webhook URL is not configured")
	}

	// Build Slack message
	msg := s.buildMessage(event)

	// Marshal to JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	logger.Debug("Sending Slack notification",
		zap.String("event_type", string(event.Type)),
	)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	// Slack returns "ok" on success
	if resp.StatusCode != http.StatusOK || string(respBody) != "ok" {
		return fmt.Errorf("Slack returned error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	logger.Debug("Slack notification sent successfully")

	return nil
}

// buildMessage builds a Slack message with rich formatting
func (s *SlackNotifier) buildMessage(event *Event) *SlackMessage {
	taskType := "Review"
	if event.TaskType == "report" {
		taskType = "Report"
	}

	// Determine if this is a success or failure event
	isSuccess := event.Type == EventReviewCompleted || event.Type == EventReportCompleted
	var emoji, color, statusText string
	if isSuccess {
		emoji = ":white_check_mark:"
		color = "good" // Green color for success
		statusText = "Completed"
	} else {
		emoji = ":x:"
		color = "danger" // Red color for failures
		statusText = "Failed"
	}

	// Build fields
	fields := []SlackField{
		{
			Title: "Repository",
			Value: event.RepoURL,
			Short: false,
		},
		{
			Title: "Task ID",
			Value: event.TaskID,
			Short: true,
		},
		{
			Title: "Time",
			Value: event.Timestamp.Format("2006-01-02 15:04:05 MST"),
			Short: true,
		},
	}

	// Add error message for failure events
	if !isSuccess && event.ErrorMessage != "" {
		fields = append(fields, SlackField{
			Title: "Error",
			Value: s.truncateText(event.ErrorMessage, 500),
			Short: false,
		})
	}

	// Add duration for success events
	if isSuccess {
		if duration, ok := event.Extra["duration_ms"].(int64); ok {
			fields = append(fields, SlackField{
				Title: "Duration",
				Value: fmt.Sprintf("%.2fs", float64(duration)/1000),
				Short: true,
			})
		}
	}

	msg := &SlackMessage{
		Text: fmt.Sprintf("%s *%s Task %s*", emoji, taskType, statusText),
		Attachments: []SlackAttachment{
			{
				Color:     color,
				Title:     fmt.Sprintf("%s Task: %s", taskType, event.TaskID),
				Fields:    fields,
				Footer:    "VerustCode Notification",
				Timestamp: event.Timestamp.Unix(),
			},
		},
	}

	// Set channel if configured
	if s.config.Channel != "" {
		msg.Channel = s.config.Channel
	}

	return msg
}

// truncateText truncates text to a maximum length
func (s *SlackNotifier) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
