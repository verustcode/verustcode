package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/pkg/logger"
)

// WebhookNotifier sends notifications via HTTP webhook
type WebhookNotifier struct {
	config *config.WebhookNotificationConfig
	client *http.Client
}

// WebhookPayload is the JSON payload sent to the webhook endpoint
type WebhookPayload struct {
	// Event type: review_failed, report_failed
	EventType string `json:"event_type"`
	// Task identifier
	TaskID string `json:"task_id"`
	// Task type: review, report
	TaskType string `json:"task_type"`
	// Repository URL
	RepoURL string `json:"repo_url"`
	// Error message that caused the failure
	ErrorMessage string `json:"error_message"`
	// Timestamp in RFC3339 format
	Timestamp string `json:"timestamp"`
	// Extra context information
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(cfg *config.WebhookNotificationConfig) *WebhookNotifier {
	return &WebhookNotifier{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the notifier name
func (w *WebhookNotifier) Name() string {
	return "webhook"
}

// Send sends a notification to the configured webhook URL
func (w *WebhookNotifier) Send(ctx context.Context, event *Event) error {
	if w.config.URL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	// Build payload
	payload := WebhookPayload{
		EventType:    string(event.Type),
		TaskID:       event.TaskID,
		TaskType:     event.TaskType,
		RepoURL:      event.RepoURL,
		ErrorMessage: event.ErrorMessage,
		Timestamp:    event.Timestamp.Format(time.RFC3339),
		Extra:        event.Extra,
	}

	// Marshal to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VerustCode-Notifier/1.0")

	// Add HMAC signature if secret is configured
	if w.config.Secret != "" {
		signature := w.computeSignature(body)
		req.Header.Set("X-VerustCode-Signature", signature)
	}

	logger.Debug("Sending webhook notification",
		zap.String("url", w.config.URL),
		zap.String("event_type", string(event.Type)),
	)

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error logging
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	logger.Debug("Webhook notification sent successfully",
		zap.Int("status_code", resp.StatusCode),
	)

	return nil
}

// computeSignature computes HMAC-SHA256 signature for the payload
func (w *WebhookNotifier) computeSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(w.config.Secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
