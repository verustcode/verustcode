// Package cursor implements the LLM Client interface for cursor-agent CLI.
// cursor-agent is the default AI agent for code review in VerustCode.
package cursor

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

// ClientName is the identifier for the Cursor client
const ClientName = "cursor"

// Default CLI command name
const defaultCLIName = "cursor-agent"

func init() {
	// Register the Cursor client factory
	llm.Register(ClientName, NewClient)
}

// Client implements the llm.Client interface for cursor-agent CLI
type Client struct {
	*llm.BaseClient
	cliPath string
}

// NewClient creates a new Cursor client
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

// Available checks if cursor-agent CLI is available
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
func (c *Client) CreateSession(ctx context.Context) (string, error) {
	c.Logger().Debug("Creating new session")

	// Build command
	cmd := exec.CommandContext(ctx, c.cliPath, "create-chat")
	c.setupCommandEnv(cmd, "")

	// Capture output
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	// Execute
	if err := cmd.Run(); err != nil {
		outputStr := output.String()
		c.Logger().Error("Failed to create session",
			zap.Error(err),
			zap.String("output", outputStr),
		)
		return "", llm.NewClientError(ClientName, "create_session", "failed to create chat", err)
	}

	sessionID := strings.TrimSpace(output.String())
	if sessionID == "" {
		return "", llm.NewClientError(ClientName, "create_session", "empty session ID returned", nil)
	}

	c.Logger().Info("Session created", zap.String("session_id", sessionID))
	return sessionID, nil
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
	// --force: skip confirmation prompts for automated execution
	args := []string{"-p", "--force", "--model", model, "--output-format", "text"}

	// Add session ID if provided
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}

	// Append extra arguments from config
	config := c.GetConfig()

	// Add API key if configured
	if config.APIKey != "" {
		args = append(args, "--api-key", config.APIKey)
	}

	if config.ExtraArgs != "" {
		extraArgs := strings.Fields(config.ExtraArgs)
		args = append(args, extraArgs...)
	}

	// Create command
	cmd := exec.CommandContext(execCtx, c.cliPath, args...)
	c.setupCommandEnv(cmd, req.WorkDir)

	// Log full command info for debugging
	apiKeyMasked := ""
	if config.APIKey != "" {
		if len(config.APIKey) > 8 {
			apiKeyMasked = config.APIKey[:4] + "..." + config.APIKey[len(config.APIKey)-4:]
		} else {
			apiKeyMasked = "***"
		}
	}

	// Build full command string for debugging (with masked sensitive args)
	maskedArgs := maskSensitiveArgs(args)
	cmdStr := c.cliPath + " " + strings.Join(maskedArgs, " ") + " < [stdin prompt]"
	reviewID := req.GetMetadata("review_id")
	c.Logger().Info("Executing cursor-agent command",
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
		return nil, llm.NewClientError(ClientName, "execute", "failed to start cursor-agent", err)
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
		c.Logger().Error("cursor-agent execution failed",
			zap.Error(err),
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderrStr),
			zap.Int("stdout_len", len(stdout.String())),
			zap.Int("stderr_len", len(stderrStr)),
		)

		// Check for authentication error (non-retryable)
		if strings.Contains(stderrStr, "Authentication required") {
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
	c.Logger().Debug("cursor-agent raw output",
		zap.String("stdout", stdout.String()),
		zap.String("stderr", stderr.String()),
		zap.Int("stdout_len", len(stdout.String())),
		zap.Int("stderr_len", len(stderr.String())),
	)

	// Build response
	resp := c.BuildResponse(stdout.String(), model, req.SessionID, req.ResponseSchema)

	// Log response details for debugging
	c.Logger().Info("cursor-agent response built",
		zap.Int("content_len", len(resp.Content)),
		zap.String("model", resp.Model),
		zap.String("session_id", resp.SessionID),
		zap.Bool("has_content", resp.Content != ""),
		// zap.String("content_full", resp.Content),
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

// doExecuteStream performs streaming execution
func (c *Client) doExecuteStream(ctx context.Context, req *llm.Request, model string, callback llm.StreamCallback) (*llm.Response, error) {
	timeout := c.GetConfig().GetTimeout(req)
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command arguments for streaming (prompt will be passed via stdin)
	// --force: skip confirmation prompts for automated execution
	args := []string{"-p", "--force", "--model", model, "--output-format", "stream-json"}

	// Add session ID if provided
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}

	// Append extra arguments from config
	config := c.GetConfig()

	// Add API key if configured
	if config.APIKey != "" {
		args = append(args, "--api-key", config.APIKey)
	}

	if config.ExtraArgs != "" {
		extraArgs := strings.Fields(config.ExtraArgs)
		args = append(args, extraArgs...)
	}

	// Create command
	cmd := exec.CommandContext(execCtx, c.cliPath, args...)
	c.setupCommandEnv(cmd, req.WorkDir)

	// Log full command info for debugging
	apiKeyMasked := ""
	if config.APIKey != "" {
		if len(config.APIKey) > 8 {
			apiKeyMasked = config.APIKey[:4] + "..." + config.APIKey[len(config.APIKey)-4:]
		} else {
			apiKeyMasked = "***"
		}
	}

	// Build full command string for debugging (with masked sensitive args)
	maskedArgs := maskSensitiveArgs(args)
	cmdStr := c.cliPath + " " + strings.Join(maskedArgs, " ") + " < [stdin prompt]"
	streamReviewID := req.GetMetadata("review_id")
	c.Logger().Info("Executing cursor-agent command (streaming)",
		zap.String("command", cmdStr),
		zap.String("review_id", streamReviewID),
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
		return nil, llm.NewClientError(ClientName, "execute_stream", "failed to start cursor-agent", err)
	}

	// Write prompt to stdin in a goroutine to avoid blocking
	go func() {
		defer stdin.Close()
		_, writeErr := stdin.Write([]byte(req.Prompt))
		if writeErr != nil {
			c.Logger().Error("Failed to write prompt to stdin (streaming)", zap.Error(writeErr))
		}
	}()

	c.Logger().Debug("Started cursor-agent streaming",
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

	// Parse JSON line and create StreamChunk
	parseJSONLine := func(line string) *llm.StreamChunk {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			return nil
		}

		msgType, _ := data["type"].(string)
		subtype, _ := data["subtype"].(string)

		// Extract session ID if present
		if chatID, ok := data["chat_id"].(string); ok && chatID != "" {
			sessionIDMu.Lock()
			extractedSessionID = chatID
			sessionIDMu.Unlock()
		}

		chunk := &llm.StreamChunk{
			Metadata: make(map[string]string),
		}

		switch msgType {
		case "system":
			chunk.Type = llm.ChunkTypeSystem
			if subtype == "init" {
				chunk.Delta = "initializing"
			}

		case "thinking":
			chunk.Type = llm.ChunkTypeThinking
			if subtype == "delta" {
				chunk.Delta = "."
			}

		case "assistant":
			chunk.Type = llm.ChunkTypeText
			// Extract text from message.content[0].text
			if message, ok := data["message"].(map[string]interface{}); ok {
				if contentArray, ok := message["content"].([]interface{}); ok && len(contentArray) > 0 {
					if firstContent, ok := contentArray[0].(map[string]interface{}); ok {
						if text, ok := firstContent["text"].(string); ok {
							chunk.Delta = text
						}
					}
				}
			}
			// Fallback to direct text field
			if chunk.Delta == "" {
				if text, ok := data["text"].(string); ok {
					chunk.Delta = text
				}
			}

		case "tool_call":
			if subtype == "started" {
				chunk.Type = llm.ChunkTypeToolCall
				chunk.Delta = "|"
				if name, ok := data["name"].(string); ok {
					chunk.ToolName = name
				}
				if input, ok := data["input"].(string); ok {
					chunk.ToolInput = input
				}
			} else if subtype == "completed" {
				chunk.Type = llm.ChunkTypeToolResult
				chunk.Delta = "|"
				if output, ok := data["output"].(string); ok {
					chunk.ToolOutput = output
				}
			}

		case "result":
			chunk.Type = llm.ChunkTypeResult
			chunk.IsComplete = true
			resultMu.Lock()
			resultReceived = true
			resultMu.Unlock()

			if result, ok := data["result"].(string); ok {
				// Find marker and trim content before it
				marker := "\n\n\n\n"
				if idx := strings.Index(result, marker); idx != -1 {
					result = result[idx+len(marker):]
				}
				chunk.Delta = result
				chunk.Content = result
			}

		case "error":
			chunk.Type = llm.ChunkTypeError
			if msg, ok := data["message"].(string); ok {
				chunk.Delta = msg
			}
		}

		return chunk
	}

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if chunk := parseJSONLine(line); chunk != nil && chunk.Delta != "" {
				outputChan <- chunk
			}
		}
		done <- true
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if chunk := parseJSONLine(line); chunk != nil && chunk.Delta != "" {
				outputChan <- chunk
			}
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
					// Result replaces the buffer
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

// setupCommandEnv sets up the command environment
func (c *Client) setupCommandEnv(cmd *exec.Cmd, workDir string) {
	// Set working directory
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Copy current environment
	cmd.Env = os.Environ()

	// Set locale
	cmd.Env = append(cmd.Env, "LANG=en_US.UTF-8")
	cmd.Env = append(cmd.Env, "LC_ALL=en_US.UTF-8")

	// Note: API key is now passed via --api-key command line argument
	// No need to set environment variable
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
