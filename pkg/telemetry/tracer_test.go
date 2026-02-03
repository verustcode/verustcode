// Package telemetry provides OpenTelemetry integration for the application.
// This file contains unit tests for the tracer functions.
package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TestTracer tests the Tracer function
func TestTracer(t *testing.T) {
	tracer := Tracer()
	if tracer == nil {
		t.Fatal("Tracer() returned nil")
	}
}

// TestStartSpan tests the StartSpan function
func TestStartSpan(t *testing.T) {
	ctx := context.Background()

	newCtx, span := StartSpan(ctx, "test-operation")
	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}
	if newCtx == nil {
		t.Fatal("StartSpan() returned nil context")
	}

	span.End()
}

// TestSpanFromContext tests the SpanFromContext function
func TestSpanFromContext(t *testing.T) {
	t.Run("with span in context", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartSpan(ctx, "test-operation")
		defer span.End()

		retrievedSpan := SpanFromContext(ctx)
		if retrievedSpan == nil {
			t.Error("SpanFromContext() returned nil for context with span")
		}
	})

	t.Run("without span in context", func(t *testing.T) {
		ctx := context.Background()
		span := SpanFromContext(ctx)
		// Should return a no-op span, not nil
		if span == nil {
			t.Error("SpanFromContext() returned nil for context without span")
		}
	})
}

// TestSetSpanError tests the SetSpanError function
func TestSetSpanError(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-operation")
	defer span.End()

	err := errors.New("test error")
	SetSpanError(span, err)

	// Verify span status is set (we can't directly check, but it shouldn't panic)
}

// TestSetSpanErrorNil tests that SetSpanError handles nil error
func TestSetSpanErrorNil(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-operation")
	defer span.End()

	// Should not panic with nil error
	SetSpanError(span, nil)
}

// TestSetSpanOK tests the SetSpanOK function
func TestSetSpanOK(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-operation")
	defer span.End()

	// Should not panic
	SetSpanOK(span)
}

// TestAddSpanEvent tests the AddSpanEvent function
func TestAddSpanEvent(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-operation")
	defer span.End()

	// Should not panic
	AddSpanEvent(span, "test-event")
	AddSpanEvent(span, "test-event-with-attrs", attribute.String("key", "value"))
}

// TestSetSpanAttributes tests the SetSpanAttributes function
func TestSetSpanAttributes(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-operation")
	defer span.End()

	// Should not panic
	SetSpanAttributes(span, attribute.String("key", "value"))
	SetSpanAttributes(span,
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
		attribute.Bool("key3", true),
	)
}

// TestAttributeKeys tests the common attribute keys
func TestAttributeKeys(t *testing.T) {
	tests := []struct {
		name string
		key  attribute.Key
		want string
	}{
		{"AttrTaskID", AttrTaskID, "task.id"},
		{"AttrTaskStatus", AttrTaskStatus, "task.status"},
		{"AttrRepoFullName", AttrRepoFullName, "repo.full_name"},
		{"AttrRepoOwner", AttrRepoOwner, "repo.owner"},
		{"AttrRepoName", AttrRepoName, "repo.name"},
		{"AttrRepoProvider", AttrRepoProvider, "repo.provider"},
		{"AttrRepoRef", AttrRepoRef, "repo.ref"},
		{"AttrReviewID", AttrReviewID, "review.id"},
		{"AttrReviewStatus", AttrReviewStatus, "review.status"},
		{"AttrReviewAgent", AttrReviewAgent, "review.agent"},
		{"AttrAgentName", AttrAgentName, "agent.name"},
		{"AttrAgentType", AttrAgentType, "agent.type"},
		{"AttrFindingsCount", AttrFindingsCount, "findings.count"},
		{"AttrDurationMs", AttrDurationMs, "duration.ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.key) != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, string(tt.key), tt.want)
			}
		})
	}
}

// TestWithTaskAttributes tests the WithTaskAttributes function
func TestWithTaskAttributes(t *testing.T) {
	opt := WithTaskAttributes("task-123", "owner/repo", "main")
	if opt == nil {
		t.Error("WithTaskAttributes() returned nil")
	}

	// Use it to create a span
	ctx := context.Background()
	_, span := StartSpan(ctx, "test", opt)
	span.End()
}

// TestWithReviewAttributes tests the WithReviewAttributes function
func TestWithReviewAttributes(t *testing.T) {
	opt := WithReviewAttributes("review-123", "cursor")
	if opt == nil {
		t.Error("WithReviewAttributes() returned nil")
	}

	// Use it to create a span
	ctx := context.Background()
	_, span := StartSpan(ctx, "test", opt)
	span.End()
}

// TestTracerName constant
func TestTracerName(t *testing.T) {
	if TracerName == "" {
		t.Error("TracerName should not be empty")
	}
	if TracerName != "github.com/verustcode/verustcode" {
		t.Errorf("TracerName = %s, want github.com/verustcode/verustcode", TracerName)
	}
}

// mockSpan is a mock implementation for testing
type mockSpan struct {
	trace.Span
	statusCode codes.Code
	events     []string
	attrs      []attribute.KeyValue
}

func (m *mockSpan) RecordError(err error, options ...trace.EventOption) {}
func (m *mockSpan) SetStatus(code codes.Code, description string)       { m.statusCode = code }
func (m *mockSpan) AddEvent(name string, options ...trace.EventOption) {
	m.events = append(m.events, name)
}
func (m *mockSpan) SetAttributes(kv ...attribute.KeyValue) { m.attrs = append(m.attrs, kv...) }
func (m *mockSpan) End(options ...trace.SpanEndOption)     {}
