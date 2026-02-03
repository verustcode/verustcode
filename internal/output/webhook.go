package output

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// DefaultWebhookTimeout is the default timeout for webhook requests (60 seconds)
	// Valid range: 30-300 seconds
	DefaultWebhookTimeout = 60

	// DefaultWebhookMaxRetries is the default maximum total number of attempts (including initial attempt)
	// Valid range: 3-12
	DefaultWebhookMaxRetries = 6

	// WebhookHeaderKey is the authentication header name
	WebhookHeaderKey = "X-SCOPEVIEW-KEY"

	// WebhookContentType is the content type for webhook requests
	WebhookContentType = "application/json"
)

// WebhookPayload represents the JSON payload sent to webhook endpoint
type WebhookPayload struct {
	// ReviewID is the unique identifier for this review
	ReviewID string `json:"review_id,omitempty"`

	// RuleID is the reviewer rule identifier
	RuleID string `json:"rule_id"`

	// RepoURL is the repository URL
	RepoURL string `json:"repo_url,omitempty"`

	// PRNumber is the PR/MR number (optional)
	PRNumber int `json:"pr_number,omitempty"`

	// Timestamp is the time when the review was completed
	Timestamp string `json:"timestamp"`

	// Data is the base64 encoded review result
	Data string `json:"data"`
}

// WebhookChannel outputs review results to a webhook endpoint
type WebhookChannel struct {
	// URL is the webhook endpoint
	url string

	// headerSecret is the value for X-SCOPEVIEW-KEY header
	headerSecret string

	// timeout is the HTTP request timeout in seconds
	timeout int

	// maxRetries is the maximum total number of attempts (including initial attempt)
	maxRetries int

	// format is the output format (json or markdown)
	// For webhooks, json is typically used for API integration
	format string

	// httpClient is the HTTP client (can be overridden for testing)
	httpClient *http.Client

	// store is the data store for logging
	store store.Store
}

// NewWebhookChannel creates a new WebhookChannel with default settings
func NewWebhookChannel(s store.Store) *WebhookChannel {
	return &WebhookChannel{
		timeout:    DefaultWebhookTimeout,
		maxRetries: DefaultWebhookMaxRetries,
		format:     "json",
		store:      s,
	}
}

// NewWebhookChannelWithConfig creates a new WebhookChannel with custom settings
func NewWebhookChannelWithConfig(url, headerSecret string, timeout, maxRetries int, format string, s store.Store) *WebhookChannel {
	if timeout <= 0 {
		timeout = DefaultWebhookTimeout
	}
	if maxRetries <= 0 {
		maxRetries = DefaultWebhookMaxRetries
	}
	if format == "" {
		format = "json"
	}

	return &WebhookChannel{
		url:          url,
		headerSecret: headerSecret,
		timeout:      timeout,
		maxRetries:   maxRetries,
		format:       format,
		store:        s,
	}
}

// Name returns the channel name
func (c *WebhookChannel) Name() string {
	return "webhook"
}

// Publish sends the review result to the webhook endpoint
func (c *WebhookChannel) Publish(ctx context.Context, result *prompt.ReviewResult, opts *PublishOptions) error {
	if c.url == "" {
		logger.Warn("Webhook channel: no URL configured, skipping")
		return nil
	}

	// Build the payload
	payload, err := c.buildPayload(result, opts)
	if err != nil {
		logger.Error("Webhook channel: failed to build payload",
			zap.String("rule_id", result.ReviewerID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to build webhook payload: %w", err)
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Webhook channel: failed to marshal payload",
			zap.String("rule_id", result.ReviewerID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create delivery log record
	deliveryLog := c.createDeliveryLog(payload, string(payloadBytes))

	// Send request with retries
	// maxRetries represents the total number of attempts (not retries after initial attempt)
	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay: 2^attempt seconds (1s, 2s, 4s, 8s, 16s, 32s)
			backoffSeconds := math.Pow(2, float64(attempt-1))
			backoffDuration := time.Duration(backoffSeconds) * time.Second

			logger.Info("Webhook channel: retrying after backoff",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoffDuration),
				zap.String("url", c.url),
			)

			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				break
			case <-time.After(backoffDuration):
				// Continue with retry
			}
		}

		// Check if context is cancelled
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			break
		}

		// Attempt to send
		err = c.sendRequest(ctx, payloadBytes)
		if err == nil {
			// Success
			deliveryLog.Status = model.WebhookStatusSuccess
			deliveryLog.RetryCount = attempt + 1 // RetryCount represents actual attempt number (1-based)
			c.saveDeliveryLog(deliveryLog)

			logger.Info("Webhook channel: successfully delivered",
				zap.String("url", c.url),
				zap.String("rule_id", result.ReviewerID),
				zap.Int("attempts", attempt+1),
			)
			return nil
		}

		lastErr = err
		deliveryLog.RetryCount = attempt + 1

		logger.Warn("Webhook channel: request failed",
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", c.maxRetries),
			zap.String("url", c.url),
			zap.Error(err),
		)
	}

	// All retries exhausted, save failure to database
	deliveryLog.Status = model.WebhookStatusFailed
	if lastErr != nil {
		deliveryLog.LastError = lastErr.Error()
	}
	c.saveDeliveryLog(deliveryLog)

	logger.Error("Webhook channel: all retries exhausted",
		zap.String("url", c.url),
		zap.String("rule_id", result.ReviewerID),
		zap.Int("total_attempts", c.maxRetries),
		zap.Error(lastErr),
	)

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", c.maxRetries, lastErr)
}

// buildPayload constructs the webhook payload from review result
func (c *WebhookChannel) buildPayload(result *prompt.ReviewResult, opts *PublishOptions) (*WebhookPayload, error) {
	// Serialize result data to JSON then base64 encode
	dataJSON, err := json.Marshal(result.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result data: %w", err)
	}
	encodedData := base64.StdEncoding.EncodeToString(dataJSON)

	ruleID := result.ReviewerID
	if ruleID == "" {
		ruleID = "unknown"
	}

	payload := &WebhookPayload{
		ReviewID:  opts.ReviewID,
		RuleID:    ruleID,
		RepoURL:   opts.RepoURL,
		PRNumber:  opts.PRNumber,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      encodedData,
	}

	return payload, nil
}

// sendRequest sends the HTTP POST request to the webhook endpoint
func (c *WebhookChannel) sendRequest(ctx context.Context, payload []byte) error {
	// Create HTTP client with timeout if not already set
	client := c.httpClient
	if client == nil {
		client = &http.Client{
			Timeout: time.Duration(c.timeout) * time.Second,
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", WebhookContentType)
	if c.headerSecret != "" {
		req.Header.Set(WebhookHeaderKey, c.headerSecret)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error message
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024)) // Limit to 1KB

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// createDeliveryLog creates a new webhook delivery log record
func (c *WebhookChannel) createDeliveryLog(payload *WebhookPayload, requestBody string) *model.ReviewResultWebhookLog {
	return &model.ReviewResultWebhookLog{
		RuleID:      payload.RuleID,
		WebhookURL:  c.url,
		RequestBody: requestBody,
		Status:      model.WebhookStatusPending,
		RetryCount:  0,
	}
}

// saveDeliveryLog saves the delivery log to database
func (c *WebhookChannel) saveDeliveryLog(log *model.ReviewResultWebhookLog) {
	if c.store == nil {
		logger.Warn("Webhook channel: store not initialized, skipping delivery log save")
		return
	}

	if err := c.store.Review().CreateWebhookLog(log); err != nil {
		logger.Error("Webhook channel: failed to save delivery log",
			zap.String("rule_id", log.RuleID),
			zap.String("url", log.WebhookURL),
			zap.String("status", string(log.Status)),
			zap.Error(err),
		)
	} else {
		logger.Debug("Webhook channel: delivery log saved",
			zap.Uint("log_id", log.ID),
			zap.String("rule_id", log.RuleID),
			zap.String("status", string(log.Status)),
		)
	}
}
