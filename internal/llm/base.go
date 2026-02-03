package llm

import (
	"context"
	"reflect"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/logger"
)

// BaseClient provides common functionality for LLM clients.
// Concrete implementations (CursorClient, GeminiClient, etc.) should embed this struct.
type BaseClient struct {
	config         *ClientConfig
	securityConfig *SecurityConfig
	schemaGen      *SchemaGenerator
	logger         *zap.Logger
}

// NewBaseClient creates a new BaseClient with the given configuration
func NewBaseClient(config *ClientConfig) *BaseClient {
	if config == nil {
		config = NewClientConfig("unknown")
	}

	return &BaseClient{
		config:         config,
		securityConfig: DefaultSecurityConfig(),
		schemaGen:      NewSchemaGenerator(),
		logger:         logger.Named("llm." + config.Name),
	}
}

// Name returns the client name
func (b *BaseClient) Name() string {
	return b.config.Name
}

// GetConfig returns the client configuration
func (b *BaseClient) GetConfig() *ClientConfig {
	return b.config
}

// SetSecurityConfig sets the security configuration
func (b *BaseClient) SetSecurityConfig(config *SecurityConfig) {
	b.securityConfig = config
}

// Logger returns the client's logger
func (b *BaseClient) Logger() *zap.Logger {
	return b.logger
}

// BuildPromptWithSchema builds a prompt with output format instructions appended
// If schema is provided, appends JSON format requirements
// If schema is nil, appends Markdown format requirements
func (b *BaseClient) BuildPromptWithSchema(prompt string, schema *ResponseSchema) string {
	if schema == nil {
		// No schema: request Markdown format output
		return prompt + MarkdownOutputPrompt()
	}
	// Has schema: request JSON format output
	schemaPrompt := BuildSchemaPrompt(schema)
	return prompt + schemaPrompt
}

// ParseResponse parses the response content according to the schema
func (b *BaseClient) ParseResponse(content string, schema *ResponseSchema) (interface{}, error) {
	if schema == nil || schema.Schema == nil {
		return nil, nil
	}

	// Get the type of the schema
	schemaType := reflect.TypeOf(schema.Schema)
	for schemaType.Kind() == reflect.Ptr {
		schemaType = schemaType.Elem()
	}

	// Create a new instance of the target type
	target := reflect.New(schemaType).Interface()

	// Parse JSON into the target
	if err := ParseResponseJSON(content, target); err != nil {
		return nil, err
	}

	return target, nil
}

// WrapPromptWithSecurity wraps the prompt with security rules
func (b *BaseClient) WrapPromptWithSecurity(prompt string) string {
	return WrapPromptWithSecurityRules(prompt, b.securityConfig)
}

// DetectPromptInjection checks for potential prompt injection
func (b *BaseClient) DetectPromptInjection(prompt string) bool {
	return DetectPromptInjection(prompt)
}

// ExecuteWithModelFn is the function signature for execution with a specific model
type ExecuteWithModelFn func(ctx context.Context, req *Request, model string) (*Response, error)

// ExecuteWithFallback executes the request with model fallback support
func (b *BaseClient) ExecuteWithFallback(ctx context.Context, req *Request, execFn ExecuteWithModelFn) (*Response, error) {
	// Get the primary model
	primaryModel := b.config.GetModel(req)

	// If no model is specified, execute once without fallback
	// This allows clients that don't require a model (e.g., qoder) to work
	if primaryModel == "" {
		b.logger.Debug("Executing without model (model not required for this client)")
		resp, err := execFn(ctx, req, "")
		return resp, err
	}

	// Try primary model
	b.logger.Debug("Executing with primary model",
		zap.String("model", primaryModel),
	)

	resp, err := execFn(ctx, req, primaryModel)
	if err == nil {
		return resp, nil
	}

	// Check if it's a model-related error
	// Guard against nil response
	respContent := ""
	if resp != nil {
		respContent = resp.Content
	}
	if !IsModelError(err, respContent) {
		return nil, err
	}

	// Get fallback models
	fallbackModels := req.GetFallbackModels()
	if len(fallbackModels) == 0 {
		return nil, err
	}

	// Try fallback models
	var lastErr error = err
	for _, fallbackModel := range fallbackModels {
		if fallbackModel == primaryModel {
			continue
		}

		b.logger.Info("Trying fallback model",
			zap.String("primary_model", primaryModel),
			zap.String("fallback_model", fallbackModel),
		)

		resp, err = execFn(ctx, req, fallbackModel)
		if err == nil {
			b.logger.Info("Fallback model succeeded",
				zap.String("model", fallbackModel),
			)
			return resp, nil
		}

		lastErr = err
		b.logger.Warn("Fallback model failed",
			zap.String("model", fallbackModel),
			zap.Error(err),
		)
	}

	return nil, NewClientError(b.config.Name, "execute",
		"all models failed (primary and fallback)", lastErr)
}

// PrepareRequest prepares the request for execution
// - Validates required fields
// - Applies default values
// - Builds prompt with schema
// - Applies security wrapper
func (b *BaseClient) PrepareRequest(req *Request) (*Request, error) {
	if req == nil {
		return nil, NewClientError(b.config.Name, "prepare", "request is nil", nil)
	}

	if req.Prompt == "" {
		return nil, NewClientError(b.config.Name, "prepare", "prompt is empty", nil)
	}

	// Create a copy to avoid modifying the original
	prepared := &Request{
		Prompt:         req.Prompt,
		Model:          req.Model,
		SessionID:      req.SessionID,
		WorkDir:        req.WorkDir,
		ResponseSchema: req.ResponseSchema,
		Options:        req.Options,
	}

	// Apply default model if not specified
	if prepared.Model == "" {
		prepared.Model = b.config.DefaultModel
	}

	// Check for prompt injection
	if b.securityConfig.EnableInjectionDetection {
		if b.DetectPromptInjection(prepared.Prompt) {
			b.logger.Warn("Potential prompt injection detected",
				zap.String("task_id", req.GetMetadata("task_id")),
			)
		}
	}

	// Build prompt with schema
	prepared.Prompt = b.BuildPromptWithSchema(prepared.Prompt, prepared.ResponseSchema)

	// Apply security wrapper
	if b.securityConfig.EnableSecurityWrapper {
		prepared.Prompt = b.WrapPromptWithSecurity(prepared.Prompt)
	}

	// Note: Prompt printing is now handled in prompt.Renderer.Render()
	// to avoid duplicate printing. Only RENDERED PROMPT will be printed.

	return prepared, nil
}

// logPreparedPrompt logs the prepared prompt content for debugging
// Outputs metadata as structured log, then prints prompt content separately for better readability
func (b *BaseClient) logPreparedPrompt(req *Request) {
	prompt := req.Prompt
	promptLen := len(prompt)

	// Structured metadata log (without prompt content)
	b.logger.Info("Prepared prompt for execution",
		zap.String("client", b.config.Name),
		zap.String("model", req.Model),
		zap.Int("prompt_length", promptLen),
		zap.String("session_id", req.SessionID),
		zap.String("work_dir", req.WorkDir),
		zap.Bool("has_schema", req.ResponseSchema != nil),
	)

	// Log prompt content for debugging
	b.logger.Debug("Prepared prompt content",
		zap.String("prompt", prompt),
	)
}

// BuildResponse builds a response with parsed data if schema is provided
func (b *BaseClient) BuildResponse(content string, model string, sessionID string, schema *ResponseSchema) *Response {
	resp := &Response{
		Content:   content,
		Model:     model,
		SessionID: sessionID,
		Metadata:  make(map[string]string),
	}

	// Parse structured data if schema is provided
	if schema != nil {
		parsed, err := b.ParseResponse(content, schema)
		resp.Parsed = parsed
		resp.ParseErr = err

		if err != nil {
			b.logger.Debug("Failed to parse response as structured data",
				zap.Error(err),
			)
		}
	}

	return resp
}

// LogRequest logs the request details
func (b *BaseClient) LogRequest(req *Request, operation string) {
	ruleID := req.GetMetadata("rule_id")
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("model", req.Model),
		zap.String("session_id", req.SessionID),
		zap.String("work_dir", req.WorkDir),
		zap.Int("prompt_length", len(req.Prompt)),
	}
	if ruleID != "" {
		fields = append(fields, zap.String("rule_id", ruleID))
	}
	b.logger.Debug("Executing request", fields...)
}

// LogResponse logs the response details
func (b *BaseClient) LogResponse(resp *Response, duration time.Duration, err error) {
	// Extract rule_id from response metadata if available
	ruleID := ""
	if resp != nil && resp.Metadata != nil {
		ruleID = resp.Metadata["rule_id"]
	}

	if err != nil {
		fields := []zap.Field{
			zap.Error(err),
			zap.Duration("duration", duration),
		}
		if ruleID != "" {
			fields = append(fields, zap.String("rule_id", ruleID))
		}
		b.logger.Error("Request failed", fields...)
		return
	}

	// Guard against nil response (should not happen if err is nil, but be defensive)
	if resp == nil {
		b.logger.Warn("Request completed with nil response",
			zap.Duration("duration", duration))
		return
	}

	fields := []zap.Field{
		zap.String("model", resp.Model),
		zap.Int("content_length", len(resp.Content)),
		zap.Duration("duration", duration),
		zap.Bool("parsed_ok", resp.ParseErr == nil),
	}
	if ruleID != "" {
		fields = append(fields, zap.String("rule_id", ruleID))
	}
	b.logger.Debug("Request completed", fields...)
}
