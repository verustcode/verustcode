// Package qoder implements the LLM Client interface for Qoder CLI.
// Qoder provides AI-powered code analysis and review capabilities.
// Official documentation: https://docs.qoder.com/zh/cli/using-cli
package qoder

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/llm"
)

// ClientName is the identifier for the Qoder client
const ClientName = "qoder"

// Default CLI command name
const defaultCLIName = "qodercli"

// Environment variable name for Qoder authentication
const qoderTokenEnvVar = "QODER_PERSONAL_ACCESS_TOKEN"

func init() {
	// Register the Qoder client factory
	llm.Register(ClientName, NewClient)
}

// Client implements the llm.Client interface for Qoder CLI
type Client struct {
	*llm.BaseClient
	cliPath string
}

// NewClient creates a new Qoder client
func NewClient(config *llm.ClientConfig) (llm.Client, error) {
	if config == nil {
		config = llm.NewClientConfig(ClientName)
	}

	// Determine CLI path
	cliPath := config.CLIPath
	if cliPath == "" {
		// Try to find in PATH
		path, err := exec.LookPath(defaultCLIName)
		if err != nil {
			cliPath = defaultCLIName // Will fail later if not found
		} else {
			cliPath = path
		}
	}

	return &Client{
		BaseClient: llm.NewBaseClient(config),
		cliPath:    cliPath,
	}, nil
}

// Available checks if Qoder CLI is available
func (c *Client) Available() bool {
	_, err := exec.LookPath(c.cliPath)
	if err != nil {
		// Try default name
		_, err = exec.LookPath(defaultCLIName)
		return err == nil
	}
	return true
}

// Execute performs a synchronous execution and returns the complete response
func (c *Client) Execute(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	startTime := time.Now()
	c.LogRequest(req, "execute")

	// Prepare the request
	prepared, err := c.PrepareRequest(req)
	if err != nil {
		return nil, err
	}

	// Execute with fallback (retry logic is handled by Executor layer)
	resp, err := c.ExecuteWithFallback(ctx, prepared, c.doExecute)

	c.LogResponse(resp, time.Since(startTime), err)
	return resp, err
}

// ExecuteStream performs a streaming execution with callback
func (c *Client) ExecuteStream(ctx context.Context, req *llm.Request, callback llm.StreamCallback) (*llm.Response, error) {
	startTime := time.Now()
	c.LogRequest(req, "execute_stream")

	// Prepare the request
	prepared, err := c.PrepareRequest(req)
	if err != nil {
		return nil, err
	}

	// Get model
	model := c.GetConfig().GetModel(prepared)

	// Execute streaming
	resp, err := c.doExecuteStream(ctx, prepared, model, callback)

	c.LogResponse(resp, time.Since(startTime), err)
	return resp, err
}

// CreateSession creates a new conversation session
// Note: Qoder CLI auto-creates sessions on first execution
// The session_id is extracted from the "init" event in stream-json mode
func (c *Client) CreateSession(ctx context.Context) (string, error) {
	c.Logger().Debug("CreateSession called - Qoder auto-creates sessions on first execution")
	// Return empty string to indicate auto-creation
	return "", nil
}

// Close releases any resources held by the client
func (c *Client) Close() error {
	// No resources to release for CLI-based client
	return nil
}

// doExecute performs the actual execution with a specific model
func (c *Client) doExecute(ctx context.Context, req *llm.Request, model string) (*llm.Response, error) {
	timeout := c.GetConfig().GetTimeout(req)
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command arguments (prompt will be passed via stdin)
	// -p: print mode for headless execution
	// --yolo: skip confirmation prompts (bypass all permission checks)
	// --output-format text: plain text output for synchronous execution
	args := []string{"-p", "--yolo", "--output-format", "text"}

	// Add session ID if provided (resume session)
	// Use --resume / -r flag as per documentation
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}

	// Append extra arguments from config
	config := c.GetConfig()
	if config.ExtraArgs != "" {
		extraArgs := strings.Fields(config.ExtraArgs)
		args = append(args, extraArgs...)
	}

	// Create command
	cmd := exec.CommandContext(execCtx, c.cliPath, args...)
	c.setupCommandEnv(cmd, req.WorkDir, config.APIKey)

	// Log full command info for debugging (with masked sensitive args)
	apiKeyMasked := maskAPIKey(config.APIKey)
	maskedArgs := maskSensitiveArgs(args)
	cmdStr := c.cliPath + " " + strings.Join(maskedArgs, " ") + " < [stdin prompt]"
	reviewID := req.GetMetadata("review_id")
	c.Logger().Info("Executing qodercli command",
		zap.String("command", cmdStr),
		zap.String("review_id", reviewID),
		zap.String("cli_path", c.cliPath),
		zap.String("work_dir", req.WorkDir),
		zap.String("model", model),
		zap.String("session_id", req.SessionID),
		zap.Strings("args", maskedArgs),
		zap.String("extra_args", config.ExtraArgs),
		zap.String("api_key", apiKeyMasked),
		zap.Bool("has_api_key", config.APIKey != ""),
	)

	// Create stdin pipe for prompt input
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, llm.NewClientError(ClientName, "execute", "failed to create stdin pipe", err)
	}

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, llm.NewClientError(ClientName, "execute", "failed to start qodercli", err)
	}

	// Write prompt to stdin in a goroutine to avoid blocking
	go func() {
		defer stdin.Close()
		promptBytes := []byte(req.Prompt)
		n, writeErr := stdin.Write(promptBytes)
		if writeErr != nil {
			c.Logger().Error("Failed to write prompt to stdin", zap.Error(writeErr))
		} else {
			c.Logger().Debug("Wrote prompt to stdin",
				zap.Int("bytes_written", n),
				zap.Int("prompt_len", len(promptBytes)),
			)
		}
	}()

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// Check for timeout
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, llm.NewClientError(ClientName, "execute", "execution timeout", llm.ErrTimeout)
		}

		stderrStr := stderr.String()
		outputStr := stdout.String() + stderrStr
		c.Logger().Error("qodercli execution failed",
			zap.Error(err),
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderrStr),
			zap.Int("stdout_len", len(stdout.String())),
			zap.Int("stderr_len", len(stderrStr)),
		)

		// Check for authentication error (non-retryable)
		if strings.Contains(stderrStr, "Authentication") ||
			strings.Contains(stderrStr, "authentication") ||
			strings.Contains(stderrStr, "token") ||
			strings.Contains(stderrStr, "unauthorized") {
			return nil, llm.NewClientError(ClientName, "execute",
				"authentication failed: "+stderrStr, err)
		}

		// Other errors are retryable
		return &llm.Response{
			Content: outputStr,
			Model:   model,
		}, llm.NewRetryableError(ClientName, "execute", "CLI execution failed: "+stderrStr, err)
	}

	// Log raw output for debugging
	c.Logger().Debug("qodercli raw output",
		zap.String("stdout", stdout.String()),
		zap.String("stderr", stderr.String()),
		zap.Int("stdout_len", len(stdout.String())),
		zap.Int("stderr_len", len(stderr.String())),
	)

	// Build response
	resp := c.BuildResponse(stdout.String(), model, req.SessionID, req.ResponseSchema)

	// Log response details for debugging
	c.Logger().Info("qodercli response built",
		zap.Int("content_len", len(resp.Content)),
		zap.String("model", resp.Model),
		zap.String("session_id", resp.SessionID),
		zap.Bool("has_content", resp.Content != ""),
	)

	// Copy metadata from request to response (including rule_id)
	if req.Options != nil && req.Options.Metadata != nil {
		if resp.Metadata == nil {
			resp.Metadata = make(map[string]string)
		}
		for k, v := range req.Options.Metadata {
			resp.Metadata[k] = v
		}
	}

	return resp, nil
}

// doExecuteStream performs streaming execution using stream-json format
func (c *Client) doExecuteStream(ctx context.Context, req *llm.Request, model string, callback llm.StreamCallback) (*llm.Response, error) {
	timeout := c.GetConfig().GetTimeout(req)
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command arguments for streaming (prompt will be passed via stdin)
	// -p: print mode for headless execution
	// --yolo: skip confirmation prompts (bypass all permission checks)
	// --output-format stream-json: streaming JSON output
	args := []string{"-p", "--yolo", "--output-format", "stream-json"}

	// Add session ID if provided (resume session)
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}

	// Append extra arguments from config
	config := c.GetConfig()
	if config.ExtraArgs != "" {
		extraArgs := strings.Fields(config.ExtraArgs)
		args = append(args, extraArgs...)
	}

	// Create command
	cmd := exec.CommandContext(execCtx, c.cliPath, args...)
	c.setupCommandEnv(cmd, req.WorkDir, config.APIKey)

	// Log full command info for debugging (with masked sensitive args)
	apiKeyMasked := maskAPIKey(config.APIKey)
	maskedArgs := maskSensitiveArgs(args)
	cmdStr := c.cliPath + " " + strings.Join(maskedArgs, " ") + " < [stdin prompt]"
	c.Logger().Info("Executing qodercli command (streaming)",
		zap.String("command", cmdStr),
		zap.String("cli_path", c.cliPath),
		zap.String("work_dir", req.WorkDir),
		zap.String("model", model),
		zap.String("session_id", req.SessionID),
		zap.Strings("args", maskedArgs),
		zap.String("extra_args", config.ExtraArgs),
		zap.String("api_key", apiKeyMasked),
		zap.Bool("has_api_key", config.APIKey != ""),
	)

	// Create stdin pipe for prompt input
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, llm.NewClientError(ClientName, "execute_stream", "failed to create stdin pipe", err)
	}

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, llm.NewClientError(ClientName, "execute_stream", "failed to create stdout pipe", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, llm.NewClientError(ClientName, "execute_stream", "failed to create stderr pipe", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, llm.NewClientError(ClientName, "execute_stream", "failed to start qodercli", err)
	}

	// Write prompt to stdin in a goroutine to avoid blocking
	go func() {
		defer stdin.Close()
		_, writeErr := stdin.Write([]byte(req.Prompt))
		if writeErr != nil {
			c.Logger().Error("Failed to write prompt to stdin (streaming)", zap.Error(writeErr))
		}
	}()

	c.Logger().Debug("Started qodercli streaming",
		zap.String("model", model),
		zap.Int("pid", cmd.Process.Pid),
	)

	// Channels for collecting output
	var textBuffer strings.Builder
	var textBufferMu sync.Mutex
	var extractedSessionID string
	var sessionIDMu sync.Mutex
	resultReceived := false
	var resultMu sync.Mutex

	// Process output in goroutines
	outputChan := make(chan *llm.StreamChunk, 100)
	done := make(chan bool, 2)

	// Read stdout - parse stream-json format
	go func() {
		scanner := bufio.NewScanner(stdout)
		// Increase buffer size for large JSON lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if chunk := c.parseStreamJSON(line, &extractedSessionID, &sessionIDMu, &resultReceived, &resultMu); chunk != nil {
				outputChan <- chunk
			}
		}
		if err := scanner.Err(); err != nil {
			c.Logger().Error("Stdout scanner error", zap.Error(err))
		}
		done <- true
	}()

	// Read stderr (for warnings/errors)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			c.Logger().Warn("Qoder stderr output", zap.String("line", line))
		}
		if err := scanner.Err(); err != nil {
			c.Logger().Error("Stderr scanner error", zap.Error(err))
		}
		done <- true
	}()

	// Process chunks and call callback
	processingDone := make(chan bool)
	go func() {
		defer func() { processingDone <- true }()
		for chunk := range outputChan {
			// Update text buffer for text chunks
			if chunk.Type == llm.ChunkTypeText || chunk.Type == llm.ChunkTypeResult {
				textBufferMu.Lock()
				if chunk.Type == llm.ChunkTypeResult {
					// Result contains final content, use it directly
					textBuffer.Reset()
					textBuffer.WriteString(chunk.Delta)
				} else {
					textBuffer.WriteString(chunk.Delta)
				}
				chunk.Content = textBuffer.String()
				textBufferMu.Unlock()
			}

			// Call callback
			if callback != nil {
				callback(chunk)
			}
		}
	}()

	// Wait for readers to finish
	<-done
	<-done
	close(outputChan)
	<-processingDone

	// Wait for command to finish
	err = cmd.Wait()

	// Get final session ID
	sessionIDMu.Lock()
	finalSessionID := extractedSessionID
	sessionIDMu.Unlock()
	if finalSessionID == "" {
		finalSessionID = req.SessionID
	}

	// Get final content
	textBufferMu.Lock()
	finalContent := textBuffer.String()
	textBufferMu.Unlock()

	// Check for errors
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, llm.NewClientError(ClientName, "execute_stream", "execution timeout", llm.ErrTimeout)
		}
		return &llm.Response{
			Content:   finalContent,
			Model:     model,
			SessionID: finalSessionID,
		}, llm.NewRetryableError(ClientName, "execute_stream", "CLI execution failed", err)
	}

	// Check if result was received
	resultMu.Lock()
	hasResult := resultReceived
	resultMu.Unlock()

	if !hasResult {
		c.Logger().Warn("Stream completed without result message")
	}

	// Build response
	resp := c.BuildResponse(finalContent, model, finalSessionID, req.ResponseSchema)

	// Copy metadata from request to response (including rule_id)
	if req.Options != nil && req.Options.Metadata != nil {
		if resp.Metadata == nil {
			resp.Metadata = make(map[string]string)
		}
		for k, v := range req.Options.Metadata {
			resp.Metadata[k] = v
		}
	}

	return resp, nil
}

// parseStreamJSON parses a line of stream-json output from qodercli
// Format based on actual qodercli output:
// - {"type":"system","subtype":"init","session_id":"...","done":false}
// - {"type":"assistant","subtype":"message","message":{"content":[{"type":"text","text":"..."}]},"session_id":"...","done":false}
// - {"type":"result","subtype":"success","message":{...},"session_id":"...","done":true}
func (c *Client) parseStreamJSON(line string, sessionID *string, sessionIDMu *sync.Mutex, resultReceived *bool, resultMu *sync.Mutex) *llm.StreamChunk {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		c.Logger().Debug("Failed to parse stream-json line", zap.String("line", line), zap.Error(err))
		return nil
	}

	msgType, _ := data["type"].(string)
	subtype, _ := data["subtype"].(string)

	chunk := &llm.StreamChunk{
		Metadata: make(map[string]string),
	}

	// Extract session_id if present (available in all message types)
	if sid, ok := data["session_id"].(string); ok && sid != "" {
		sessionIDMu.Lock()
		*sessionID = sid
		sessionIDMu.Unlock()
		chunk.Metadata["session_id"] = sid
		c.Logger().Debug("Extracted session ID", zap.String("session_id", sid))
	}

	switch msgType {
	case "system":
		// System message, typically init
		chunk.Type = llm.ChunkTypeSystem
		if subtype == "init" {
			chunk.Delta = "initializing"
			// Log available tools for debugging
			if tools, ok := data["tools"].([]interface{}); ok {
				toolNames := make([]string, len(tools))
				for i, t := range tools {
					if name, ok := t.(string); ok {
						toolNames[i] = name
					}
				}
				c.Logger().Debug("Qoder tools available", zap.Strings("tools", toolNames))
			}
		}

	case "assistant":
		// Assistant message containing response text
		chunk.Type = llm.ChunkTypeText
		chunk.Delta = c.extractTextFromMessage(data)

	case "result":
		// Final result message
		chunk.Type = llm.ChunkTypeResult
		chunk.IsComplete = true
		resultMu.Lock()
		*resultReceived = true
		resultMu.Unlock()

		// Extract final text from result message
		text := c.extractTextFromMessage(data)
		chunk.Content = text
		chunk.Delta = text

		// Log result status
		if subtype != "" {
			chunk.Metadata["status"] = subtype
		}

		// Extract usage stats if available
		if message, ok := data["message"].(map[string]interface{}); ok {
			if usage, ok := message["usage"].(map[string]interface{}); ok {
				if usageJSON, err := json.Marshal(usage); err == nil {
					chunk.Metadata["usage"] = string(usageJSON)
				}
			}
		}

	case "tool_use", "tool_call":
		// Tool call (if qoder reports tool usage)
		chunk.Type = llm.ChunkTypeToolCall
		if name, ok := data["tool_name"].(string); ok {
			chunk.ToolName = name
		} else if name, ok := data["name"].(string); ok {
			chunk.ToolName = name
		}
		chunk.Delta = "|"

	case "tool_result":
		// Tool result
		chunk.Type = llm.ChunkTypeToolResult
		if output, ok := data["output"].(string); ok {
			chunk.ToolOutput = output
		}
		chunk.Delta = "|"

	case "error":
		// Error message
		chunk.Type = llm.ChunkTypeError
		if msg, ok := data["message"].(string); ok {
			chunk.Delta = msg
		} else if errStr, ok := data["error"].(string); ok {
			chunk.Delta = errStr
		}

	default:
		// Unknown type, try to extract any text content
		if text := c.extractTextFromMessage(data); text != "" {
			chunk.Type = llm.ChunkTypeText
			chunk.Delta = text
		} else {
			c.Logger().Debug("Unknown message type", zap.String("type", msgType), zap.String("subtype", subtype))
			return nil
		}
	}

	return chunk
}

// extractTextFromMessage extracts text content from a qodercli message
// Message format: {"message": {"content": [{"type": "text", "text": "..."}]}}
func (c *Client) extractTextFromMessage(data map[string]interface{}) string {
	message, ok := data["message"].(map[string]interface{})
	if !ok {
		return ""
	}

	content, ok := message["content"].([]interface{})
	if !ok {
		return ""
	}

	var textParts []string
	for _, item := range content {
		if contentItem, ok := item.(map[string]interface{}); ok {
			// Check for text type content
			if itemType, _ := contentItem["type"].(string); itemType == "text" {
				if text, ok := contentItem["text"].(string); ok && text != "" {
					textParts = append(textParts, text)
				}
			}
		}
	}

	return strings.Join(textParts, "")
}

// setupCommandEnv sets up the command environment
func (c *Client) setupCommandEnv(cmd *exec.Cmd, workDir string, apiKey string) {
	// Set working directory
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Copy current environment
	cmd.Env = os.Environ()

	// Set locale
	cmd.Env = append(cmd.Env, "LANG=en_US.UTF-8")
	cmd.Env = append(cmd.Env, "LC_ALL=en_US.UTF-8")

	// Set Qoder API key via environment variable
	// Qoder uses QODER_PERSONAL_ACCESS_TOKEN for authentication
	if apiKey != "" {
		cmd.Env = append(cmd.Env, qoderTokenEnvVar+"="+apiKey)
		c.Logger().Debug("Set Qoder API key via environment variable",
			zap.String("env_var", qoderTokenEnvVar),
			zap.String("api_key_masked", maskAPIKey(apiKey)),
		)
	}
}

// maskAPIKey masks the API key for logging
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) > 8 {
		return key[:4] + "..." + key[len(key)-4:]
	}
	return "***"
}

// isSensitiveFlag checks if the given flag is a sensitive parameter flag
func isSensitiveFlag(flag string) bool {
	sensitiveFlags := []string{"--api-key", "--token", "--secret", "--password"}
	for _, sf := range sensitiveFlags {
		if flag == sf {
			return true
		}
	}
	return false
}

// maskSensitiveArgs masks sensitive values in command line arguments
// It identifies sensitive flags like --api-key, --token, --secret, --password
// and masks their corresponding values
func maskSensitiveArgs(args []string) []string {
	masked := make([]string, len(args))
	copy(masked, args)

	for i := 0; i < len(masked)-1; i++ {
		if isSensitiveFlag(masked[i]) && i+1 < len(masked) {
			masked[i+1] = maskAPIKey(masked[i+1])
		}
	}
	return masked
}
