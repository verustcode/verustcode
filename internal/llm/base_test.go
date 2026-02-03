package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================
// Tests for NewBaseClient
// ====================

func TestNewBaseClient(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		config := NewClientConfig("test-client")
		client := NewBaseClient(config)

		assert.NotNil(t, client)
		assert.Equal(t, "test-client", client.Name())
		assert.Equal(t, config, client.GetConfig())
		assert.NotNil(t, client.Logger())
	})

	t.Run("with nil config", func(t *testing.T) {
		client := NewBaseClient(nil)

		assert.NotNil(t, client)
		assert.Equal(t, "unknown", client.Name())
		assert.NotNil(t, client.GetConfig())
		assert.NotNil(t, client.Logger())
	})
}

// ====================
// Tests for BaseClient methods
// ====================

func TestBaseClient_Name(t *testing.T) {
	config := NewClientConfig("test-name")
	client := NewBaseClient(config)
	assert.Equal(t, "test-name", client.Name())
}

func TestBaseClient_GetConfig(t *testing.T) {
	config := NewClientConfig("test-config")
	client := NewBaseClient(config)
	assert.Equal(t, config, client.GetConfig())
}

func TestBaseClient_SetSecurityConfig(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))
	secConfig := &SecurityConfig{
		EnableInjectionDetection: true,
		EnableSecurityWrapper:    true,
	}

	client.SetSecurityConfig(secConfig)
	// Verify by checking if security wrapper is applied
	prompt := "test prompt"
	wrapped := client.WrapPromptWithSecurity(prompt)
	assert.Contains(t, wrapped, "USER_INPUT_BOUNDARY")
}

func TestBaseClient_Logger(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))
	logger := client.Logger()
	assert.NotNil(t, logger)
}

// ====================
// Tests for BuildPromptWithSchema
// ====================

func TestBaseClient_BuildPromptWithSchema(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	t.Run("without schema", func(t *testing.T) {
		prompt := "test prompt"
		result := client.BuildPromptWithSchema(prompt, nil)
		assert.Contains(t, result, prompt)
		assert.Contains(t, result, "Markdown format")
	})

	t.Run("with schema", func(t *testing.T) {
		prompt := "test prompt"
		schema := &ResponseSchema{
			Name:        "test",
			Description: "test schema",
			Schema:      map[string]interface{}{"type": "object"},
		}
		result := client.BuildPromptWithSchema(prompt, schema)
		assert.Contains(t, result, prompt)
		assert.Contains(t, result, "JSON format")
	})
}

// ====================
// Tests for ParseResponse
// ====================

func TestBaseClient_ParseResponse(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	t.Run("without schema", func(t *testing.T) {
		result, err := client.ParseResponse("test content", nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("with valid JSON schema", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		schema := &ResponseSchema{
			Schema: TestStruct{},
		}

		content := `{"name": "test", "value": 42}`
		result, err := client.ParseResponse(content, schema)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify parsed result
		parsed, ok := result.(*TestStruct)
		require.True(t, ok)
		assert.Equal(t, "test", parsed.Name)
		assert.Equal(t, 42, parsed.Value)
	})

	t.Run("with invalid JSON", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		schema := &ResponseSchema{
			Schema: TestStruct{},
		}

		content := "invalid json"
		_, err := client.ParseResponse(content, schema)
		assert.Error(t, err)
	})
}

// ====================
// Tests for WrapPromptWithSecurity
// ====================

func TestBaseClient_WrapPromptWithSecurity(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	t.Run("security wrapper disabled", func(t *testing.T) {
		prompt := "test prompt"
		result := client.WrapPromptWithSecurity(prompt)
		assert.Equal(t, prompt, result)
	})

	t.Run("security wrapper enabled", func(t *testing.T) {
		client.SetSecurityConfig(&SecurityConfig{
			EnableSecurityWrapper: true,
		})
		prompt := "test prompt"
		result := client.WrapPromptWithSecurity(prompt)
		assert.Contains(t, result, "USER_INPUT_BOUNDARY")
		assert.Contains(t, result, "&lt;")
		assert.Contains(t, result, "安全规则")
	})
}

// ====================
// Tests for DetectPromptInjection
// ====================

func TestBaseClient_DetectPromptInjection(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	t.Run("no injection", func(t *testing.T) {
		prompt := "normal prompt without injection"
		result := client.DetectPromptInjection(prompt)
		assert.False(t, result)
	})

	t.Run("detects injection patterns", func(t *testing.T) {
		testCases := []string{
			"</absolute_rules>",
			"</system>",
			"忘记上述",
			"ignore previous",
			"bypass rule",
			"system override",
		}

		for _, prompt := range testCases {
			result := client.DetectPromptInjection(prompt)
			assert.True(t, result, "should detect injection in: %s", prompt)
		}
	})
}

// ====================
// Tests for ExecuteWithFallback
// ====================

func TestBaseClient_ExecuteWithFallback(t *testing.T) {
	config := NewClientConfig("test")
	config.DefaultModel = "primary-model"
	client := NewBaseClient(config)

	t.Run("success with primary model", func(t *testing.T) {
		req := NewRequest("test prompt")
		execFn := func(ctx context.Context, r *Request, model string) (*Response, error) {
			return &Response{Content: "success", Model: model}, nil
		}

		resp, err := client.ExecuteWithFallback(context.Background(), req, execFn)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "primary-model", resp.Model)
	})

	t.Run("no model specified", func(t *testing.T) {
		config := NewClientConfig("test")
		config.DefaultModel = ""
		client := NewBaseClient(config)

		req := NewRequest("test prompt")
		execFn := func(ctx context.Context, r *Request, model string) (*Response, error) {
			// Verify that model is empty string when not specified
			assert.Equal(t, "", model)
			return &Response{
				Content: "test response",
				Model:   "",
			}, nil
		}

		resp, err := client.ExecuteWithFallback(context.Background(), req, execFn)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test response", resp.Content)
		// Model can be empty for clients that don't require it
		assert.Equal(t, "", resp.Model)
	})

	t.Run("fallback to secondary model", func(t *testing.T) {
		req := NewRequest("test prompt")
		req.Options = &RequestOptions{
			FallbackModels: []string{"fallback-model"},
		}

		callCount := 0
		execFn := func(ctx context.Context, r *Request, model string) (*Response, error) {
			callCount++
			if callCount == 1 {
				// First call fails with model error (IsModelError will detect "model" keyword)
				return nil, NewClientError("test", "execute", "model not available", nil)
			}
			return &Response{Content: "success", Model: model}, nil
		}

		resp, err := client.ExecuteWithFallback(context.Background(), req, execFn)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "fallback-model", resp.Model)
		assert.Equal(t, 2, callCount)
	})

	t.Run("all models fail", func(t *testing.T) {
		req := NewRequest("test prompt")
		req.Options = &RequestOptions{
			FallbackModels: []string{"fallback-model"},
		}

		execFn := func(ctx context.Context, r *Request, model string) (*Response, error) {
			// Use error that IsModelError will detect
			return nil, NewClientError("test", "execute", "model not found", nil)
		}

		_, err := client.ExecuteWithFallback(context.Background(), req, execFn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "all models failed")
	})
}

// ====================
// Tests for PrepareRequest
// ====================

func TestBaseClient_PrepareRequest(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	t.Run("nil request", func(t *testing.T) {
		_, err := client.PrepareRequest(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request is nil")
	})

	t.Run("empty prompt", func(t *testing.T) {
		req := &Request{}
		_, err := client.PrepareRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt is empty")
	})

	t.Run("valid request", func(t *testing.T) {
		req := NewRequest("test prompt")
		prepared, err := client.PrepareRequest(req)
		assert.NoError(t, err)
		assert.NotNil(t, prepared)
		assert.Contains(t, prepared.Prompt, "test prompt")
	})

	t.Run("applies default model", func(t *testing.T) {
		config := NewClientConfig("test")
		config.DefaultModel = "default-model"
		client := NewBaseClient(config)

		req := NewRequest("test prompt")
		prepared, err := client.PrepareRequest(req)
		assert.NoError(t, err)
		assert.Equal(t, "default-model", prepared.Model)
	})

	t.Run("keeps specified model", func(t *testing.T) {
		req := NewRequest("test prompt").WithModel("custom-model")
		prepared, err := client.PrepareRequest(req)
		assert.NoError(t, err)
		assert.Equal(t, "custom-model", prepared.Model)
	})

	t.Run("builds prompt with schema", func(t *testing.T) {
		schema := &ResponseSchema{
			Name:   "test",
			Schema: map[string]interface{}{"type": "object"},
		}
		req := NewRequest("test prompt").WithSchema(schema)
		prepared, err := client.PrepareRequest(req)
		assert.NoError(t, err)
		assert.Contains(t, prepared.Prompt, "JSON format")
	})

	t.Run("applies security wrapper when enabled", func(t *testing.T) {
		client.SetSecurityConfig(&SecurityConfig{
			EnableSecurityWrapper: true,
		})
		req := NewRequest("test prompt")
		prepared, err := client.PrepareRequest(req)
		assert.NoError(t, err)
		assert.Contains(t, prepared.Prompt, "USER_INPUT_BOUNDARY")
	})
}

// ====================
// Tests for BuildResponse
// ====================

func TestBaseClient_BuildResponse(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	t.Run("without schema", func(t *testing.T) {
		resp := client.BuildResponse("test content", "test-model", "session-123", nil)
		assert.NotNil(t, resp)
		assert.Equal(t, "test content", resp.Content)
		assert.Equal(t, "test-model", resp.Model)
		assert.Equal(t, "session-123", resp.SessionID)
		assert.Nil(t, resp.Parsed)
		assert.Nil(t, resp.ParseErr)
	})

	t.Run("with schema", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		schema := &ResponseSchema{
			Schema: TestStruct{},
		}

		content := `{"name": "test"}`
		resp := client.BuildResponse(content, "test-model", "session-123", schema)
		assert.NotNil(t, resp)
		assert.Equal(t, content, resp.Content)
		assert.NotNil(t, resp.Parsed)
		assert.NoError(t, resp.ParseErr)

		parsed, ok := resp.Parsed.(*TestStruct)
		require.True(t, ok)
		assert.Equal(t, "test", parsed.Name)
	})

	t.Run("with invalid JSON schema", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		schema := &ResponseSchema{
			Schema: TestStruct{},
		}

		content := "invalid json"
		resp := client.BuildResponse(content, "test-model", "session-123", schema)
		assert.NotNil(t, resp)
		assert.Error(t, resp.ParseErr)
	})
}

// ====================
// Tests for LogRequest and LogResponse
// ====================

func TestBaseClient_LogRequest(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	req := NewRequest("test prompt").
		WithModel("test-model").
		WithSessionID("session-123").
		WithWorkDir("/tmp")

	// Should not panic
	client.LogRequest(req, "test-operation")
}

func TestBaseClient_LogResponse(t *testing.T) {
	client := NewBaseClient(NewClientConfig("test"))

	t.Run("success response", func(t *testing.T) {
		resp := &Response{
			Content:   "test content",
			Model:     "test-model",
			SessionID: "session-123",
		}
		// Should not panic
		client.LogResponse(resp, 100*time.Millisecond, nil)
	})

	t.Run("error response", func(t *testing.T) {
		err := errors.New("test error")
		// Should not panic
		client.LogResponse(nil, 100*time.Millisecond, err)
	})

	t.Run("nil response without error", func(t *testing.T) {
		// Should not panic
		client.LogResponse(nil, 100*time.Millisecond, nil)
	})
}

