package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/pkg/logger"
)

// EmailNotifier sends notifications via SMTP email
type EmailNotifier struct {
	config *config.EmailNotificationConfig
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(cfg *config.EmailNotificationConfig) *EmailNotifier {
	return &EmailNotifier{
		config: cfg,
	}
}

// Name returns the notifier name
func (e *EmailNotifier) Name() string {
	return "email"
}

// Send sends an email notification
func (e *EmailNotifier) Send(ctx context.Context, event *Event) error {
	// Validate configuration
	if e.config.SMTPHost == "" {
		return fmt.Errorf("SMTP host is not configured")
	}
	if len(e.config.To) == 0 {
		return fmt.Errorf("no recipient email addresses configured")
	}
	if e.config.From == "" {
		return fmt.Errorf("sender email address is not configured")
	}

	// Build email content
	subject := e.buildSubject(event)
	body := e.buildBody(event)

	// Build email message
	msg := e.buildMessage(subject, body)

	// Send email
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	logger.Debug("Sending email notification",
		zap.String("smtp_host", e.config.SMTPHost),
		zap.Int("smtp_port", e.config.SMTPPort),
		zap.Strings("to", e.config.To),
	)

	var auth smtp.Auth
	if e.config.Username != "" && e.config.Password != "" {
		auth = smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPHost)
	}

	// Try to send with TLS first (port 465 or 587)
	var err error
	if e.config.SMTPPort == 465 {
		err = e.sendWithTLS(addr, auth, msg)
	} else {
		err = smtp.SendMail(addr, auth, e.config.From, e.config.To, []byte(msg))
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Debug("Email notification sent successfully",
		zap.Strings("recipients", e.config.To),
	)

	return nil
}

// sendWithTLS sends email over TLS (for port 465)
func (e *EmailNotifier) sendWithTLS(addr string, auth smtp.Auth, msg string) error {
	// Connect with TLS
	tlsConfig := &tls.Config{
		ServerName: e.config.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate if configured
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(e.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, to := range e.config.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", to, err)
		}
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// buildSubject builds the email subject line
func (e *EmailNotifier) buildSubject(event *Event) string {
	taskType := "Review"
	if event.TaskType == "report" {
		taskType = "Report"
	}

	// Determine if this is a success or failure event
	isSuccess := event.Type == EventReviewCompleted || event.Type == EventReportCompleted
	status := "Failed"
	if isSuccess {
		status = "Completed"
	}

	return fmt.Sprintf("[VerustCode] %s Task %s: %s", taskType, status, event.TaskID)
}

// buildBody builds the email body
func (e *EmailNotifier) buildBody(event *Event) string {
	var sb strings.Builder

	taskType := "Review"
	if event.TaskType == "report" {
		taskType = "Report"
	}

	// Determine if this is a success or failure event
	isSuccess := event.Type == EventReviewCompleted || event.Type == EventReportCompleted
	status := "Failed"
	if isSuccess {
		status = "Completed"
	}

	sb.WriteString(fmt.Sprintf("%s Task %s\n", taskType, status))
	sb.WriteString("================================\n\n")
	sb.WriteString(fmt.Sprintf("Task ID: %s\n", event.TaskID))
	sb.WriteString(fmt.Sprintf("Repository: %s\n", event.RepoURL))
	sb.WriteString(fmt.Sprintf("Time: %s\n", event.Timestamp.Format("2006-01-02 15:04:05 MST")))

	// Add error message for failure events
	if !isSuccess && event.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("\nError:\n%s\n", event.ErrorMessage))
	}

	// Add duration for success events
	if isSuccess {
		if duration, ok := event.Extra["duration_ms"].(int64); ok {
			sb.WriteString(fmt.Sprintf("\nDuration: %.2f seconds\n", float64(duration)/1000))
		}
	}

	if len(event.Extra) > 0 {
		sb.WriteString("\nAdditional Information:\n")
		for k, v := range event.Extra {
			// Skip duration_ms as it's already displayed above
			if k == "duration_ms" {
				continue
			}
			sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}

	sb.WriteString("\n--\nSent by VerustCode Notification System\n")

	return sb.String()
}

// buildMessage builds the complete SMTP message with headers
func (e *EmailNotifier) buildMessage(subject, body string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("From: %s\r\n", e.config.From))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.config.To, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)

	return sb.String()
}
