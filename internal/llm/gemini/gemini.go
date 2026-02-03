// Package gemini implements the LLM Client interface for Gemini CLI.
// Gemini CLI provides Google's AI-powered code analysis capabilities.
package gemini

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

// ClientName is the identifier for the Gemini client
const ClientName = "gemini"

// Default CLI command name
const defaultCLIName = "gemini"

func init() {
	// Register the Gemini client factory
	llm.Register(ClientName, NewClient)
}

// Client implements the llm.Client interface for Gemini CLI
type Client struct {
	*llm.BaseClient
	cliPath string
}

// NewClient creates a new Gemini client
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

// Available checks if Gemini CLI is available
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
// Note: Gemini CLI auto-creates sessions on first execution
// The session_id is extracted from the "init" event in stream-json mode
func (c *Client) CreateSession(ctx context.Context) (string, error) {
	c.Logger().Debug("CreateSession called - Gemini auto-creates sessions on first execution")

	// Return empty string to indicate auto-creation
	// The actual session ID will be extracted from the first execution's init event
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
	args := []string{"-p", "--model", model, "--output-format", "text", "--yolo"}

	// Add session ID if provided (resume session)
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

	// Log full command info for debugging (with masked sensitive args)
	apiKeyMasked := maskAPIKey(config.APIKey)
	maskedArgs := maskSensitiveArgs(args)
	cmdStr := c.cliPath + " " + strings.Join(maskedArgs, " ") + " < [stdin prompt]"
	c.Logger().Info("Executing gemini command",
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
		return nil, llm.NewClientError(ClientName, "execute", "failed to create stdin pipe", err)
	}

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, llm.NewClientError(ClientName, "execute", "failed to start gemini", err)
	}

	// Write prompt to stdin in a goroutine to avoid blocking
	go func() {
		defer stdin.Close()
		_, writeErr := stdin.Write([]byte(req.Prompt))
		if writeErr != nil {
			c.Logger().Error("Failed to write prompt to stdin", zap.Error(writeErr))
		}
	}()

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// Check for timeout
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, llm.NewClientError(ClientName, "execute", "execution timeout", llm.ErrTimeout)
		}

		outputStr := stdout.String() + stderr.String()
		return &llm.Response{
			Content: outputStr,
			Model:   model,
		}, llm.NewRetryableError(ClientName, "execute", "CLI execution failed: "+stderr.String(), err)
	}

	// Build response
	resp := c.BuildResponse(stdout.String(), model, req.SessionID, req.ResponseSchema)

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
	args := []string{"-p", "--model", model, "--output-format", "stream-json", "--yolo"}

	// Add session ID if provided (resume session)
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

	// Log full command info for debugging (with masked sensitive args)
	apiKeyMasked := maskAPIKey(config.APIKey)
	maskedArgs := maskSensitiveArgs(args)
	cmdStr := c.cliPath + " " + strings.Join(maskedArgs, " ") + " < [stdin prompt]"
	c.Logger().Info("Executing gemini command (streaming)",
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
		return nil, llm.NewClientError(ClientName, "execute_stream", "failed to start gemini", err)
	}

	// Write prompt to stdin in a goroutine to avoid blocking
	go func() {
		defer stdin.Close()
		_, writeErr := stdin.Write([]byte(req.Prompt))
		if writeErr != nil {
			c.Logger().Error("Failed to write prompt to stdin (streaming)", zap.Error(writeErr))
		}
	}()

	c.Logger().Debug("Started gemini streaming",
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
			c.Logger().Debug("Failed to parse JSON line", zap.String("line", line), zap.Error(err))
			return nil
		}

		msgType, _ := data["type"].(string)
		chunk := &llm.StreamChunk{
			Metadata: make(map[string]string),
		}

		switch msgType {
		case "init":
			// Extract session_id from init event
			if sid, ok := data["session_id"].(string); ok && sid != "" {
				sessionIDMu.Lock()
				extractedSessionID = sid
				sessionIDMu.Unlock()
				chunk.Metadata["session_id"] = sid
				c.Logger().Debug("Extracted session ID from init event", zap.String("session_id", sid))
			}
			chunk.Type = llm.ChunkTypeSystem
			chunk.Delta = "initializing"

		case "message":
			role, _ := data["role"].(string)
			content, _ := data["content"].(string)
			delta, _ := data["delta"].(bool)

			if role == "assistant" {
				chunk.Type = llm.ChunkTypeText
				chunk.Delta = content

				// Accumulate text if delta mode
				if delta {
					textBufferMu.Lock()
					textBuffer.WriteString(content)
					chunk.Content = textBuffer.String()
					textBufferMu.Unlock()
				} else {
					// Non-delta message, use content directly
					chunk.Content = content
				}
			} else if role == "user" {
				// User message, skip or log
				chunk.Type = llm.ChunkTypeSystem
				chunk.Delta = ""
			}

		case "tool_use":
			chunk.Type = llm.ChunkTypeToolCall
			if name, ok := data["tool_name"].(string); ok {
				chunk.ToolName = name
			}
			if toolID, ok := data["tool_id"].(string); ok {
				chunk.Metadata["tool_id"] = toolID
			}
			if params, ok := data["parameters"].(map[string]interface{}); ok {
				if paramsJSON, err := json.Marshal(params); err == nil {
					chunk.ToolInput = string(paramsJSON)
				}
			}
			chunk.Delta = "|"

		case "tool_result":
			chunk.Type = llm.ChunkTypeToolResult
			if toolID, ok := data["tool_id"].(string); ok {
				chunk.Metadata["tool_id"] = toolID
			}
			if output, ok := data["output"].(string); ok {
				chunk.ToolOutput = output
			}
			if status, ok := data["status"].(string); ok {
				chunk.Metadata["status"] = status
			}
			chunk.Delta = "|"

		case "result":
			chunk.Type = llm.ChunkTypeResult
			chunk.IsComplete = true
			resultMu.Lock()
			resultReceived = true
			resultMu.Unlock()

			// Use accumulated text as final content
			textBufferMu.Lock()
			finalContent := textBuffer.String()
			textBufferMu.Unlock()

			chunk.Content = finalContent
			chunk.Delta = finalContent

			// Extract stats if available
			if stats, ok := data["stats"].(map[string]interface{}); ok {
				if statsJSON, err := json.Marshal(stats); err == nil {
					chunk.Metadata["stats"] = string(statsJSON)
				}
			}
			if status, ok := data["status"].(string); ok {
				chunk.Metadata["status"] = status
			}

		default:
			// Unknown type, log and skip
			c.Logger().Debug("Unknown message type", zap.String("type", msgType))
			return nil
		}

		return chunk
	}

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if chunk := parseJSONLine(line); chunk != nil && (chunk.Delta != "" || chunk.IsComplete) {
				outputChan <- chunk
			}
		}
		if err := scanner.Err(); err != nil {
			c.Logger().Error("Stdout scanner error", zap.Error(err))
		}
		done <- true
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			c.Logger().Warn("Gemini stderr output", zap.String("line", line))
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
