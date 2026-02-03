package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ====================
// Tests for NewRequest
// ====================

func TestNewRequest(t *testing.T) {
	req := NewRequest("test prompt")
	assert.NotNil(t, req)
	assert.Equal(t, "test prompt", req.Prompt)
	assert.Empty(t, req.Model)
	assert.Empty(t, req.SessionID)
	assert.Empty(t, req.WorkDir)
	assert.Nil(t, req.ResponseSchema)
	assert.Nil(t, req.Options)
}

// ====================
// Tests for Request builder methods
// ====================

func TestRequest_WithModel(t *testing.T) {
	req := NewRequest("test").WithModel("test-model")
	assert.Equal(t, "test-model", req.Model)
}

func TestRequest_WithSessionID(t *testing.T) {
	req := NewRequest("test").WithSessionID("session-123")
	assert.Equal(t, "session-123", req.SessionID)
}

func TestRequest_WithWorkDir(t *testing.T) {
	req := NewRequest("test").WithWorkDir("/tmp/test")
	assert.Equal(t, "/tmp/test", req.WorkDir)
}

func TestRequest_WithSchema(t *testing.T) {
	schema := &ResponseSchema{
		Name:   "test",
		Schema: map[string]interface{}{"type": "object"},
	}
	req := NewRequest("test").WithSchema(schema)
	assert.Equal(t, schema, req.ResponseSchema)
}

func TestRequest_WithOptions(t *testing.T) {
	opts := &RequestOptions{
		Timeout: 30 * time.Second,
		MaxRetries: 3,
	}
	req := NewRequest("test").WithOptions(opts)
	assert.Equal(t, opts, req.Options)
}

func TestRequest_Chaining(t *testing.T) {
	schema := &ResponseSchema{Name: "test"}
	opts := &RequestOptions{Timeout: 10 * time.Second}

	req := NewRequest("test prompt").
		WithModel("test-model").
		WithSessionID("session-123").
		WithWorkDir("/tmp").
		WithSchema(schema).
		WithOptions(opts)

	assert.Equal(t, "test prompt", req.Prompt)
	assert.Equal(t, "test-model", req.Model)
	assert.Equal(t, "session-123", req.SessionID)
	assert.Equal(t, "/tmp", req.WorkDir)
	assert.Equal(t, schema, req.ResponseSchema)
	assert.Equal(t, opts, req.Options)
}

// ====================
// Tests for GetTimeout
// ====================

func TestRequest_GetTimeout(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		req := NewRequest("test")
		timeout := req.GetTimeout(5 * time.Second)
		assert.Equal(t, 5*time.Second, timeout)
	})

	t.Run("with timeout in options", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			Timeout: 10 * time.Second,
		})
		timeout := req.GetTimeout(5 * time.Second)
		assert.Equal(t, 10*time.Second, timeout)
	})

	t.Run("zero timeout uses default", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			Timeout: 0,
		})
		timeout := req.GetTimeout(5 * time.Second)
		assert.Equal(t, 5*time.Second, timeout)
	})
}

// ====================
// Tests for GetMaxRetries
// ====================

func TestRequest_GetMaxRetries(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		req := NewRequest("test")
		retries := req.GetMaxRetries(3)
		assert.Equal(t, 3, retries)
	})

	t.Run("with max retries in options", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			MaxRetries: 5,
		})
		retries := req.GetMaxRetries(3)
		assert.Equal(t, 5, retries)
	})

	t.Run("zero retries uses default", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			MaxRetries: 0,
		})
		retries := req.GetMaxRetries(3)
		assert.Equal(t, 3, retries)
	})
}

// ====================
// Tests for GetRetryDelay
// ====================

func TestRequest_GetRetryDelay(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		req := NewRequest("test")
		delay := req.GetRetryDelay(1 * time.Second)
		assert.Equal(t, 1*time.Second, delay)
	})

	t.Run("with retry delay in options", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			RetryDelay: 2 * time.Second,
		})
		delay := req.GetRetryDelay(1 * time.Second)
		assert.Equal(t, 2*time.Second, delay)
	})

	t.Run("zero delay uses default", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			RetryDelay: 0,
		})
		delay := req.GetRetryDelay(1 * time.Second)
		assert.Equal(t, 1*time.Second, delay)
	})
}

// ====================
// Tests for GetFallbackModels
// ====================

func TestRequest_GetFallbackModels(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		req := NewRequest("test")
		models := req.GetFallbackModels()
		assert.Nil(t, models)
	})

	t.Run("with fallback models", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			FallbackModels: []string{"model1", "model2"},
		})
		models := req.GetFallbackModels()
		assert.Equal(t, []string{"model1", "model2"}, models)
	})

	t.Run("empty fallback models", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			FallbackModels: []string{},
		})
		models := req.GetFallbackModels()
		assert.Equal(t, []string{}, models)
	})
}

// ====================
// Tests for GetMetadata
// ====================

func TestRequest_GetMetadata(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		req := NewRequest("test")
		value := req.GetMetadata("key")
		assert.Empty(t, value)
	})

	t.Run("no metadata in options", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{})
		value := req.GetMetadata("key")
		assert.Empty(t, value)
	})

	t.Run("with metadata", func(t *testing.T) {
		req := NewRequest("test").WithOptions(&RequestOptions{
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		})
		assert.Equal(t, "value1", req.GetMetadata("key1"))
		assert.Equal(t, "value2", req.GetMetadata("key2"))
		assert.Empty(t, req.GetMetadata("nonexistent"))
	})
}

