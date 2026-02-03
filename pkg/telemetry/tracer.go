// Package telemetry provides OpenTelemetry integration for the application.
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the default tracer name for the application
	TracerName = "github.com/verustcode/verustcode"
)

// Tracer returns the global tracer for the application
func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// StartSpan starts a new span with the given name and returns the context and span.
// The caller is responsible for calling span.End() when the operation is complete.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from the context.
// If no span is found, a no-op span is returned.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// SetSpanError records an error on the span and sets its status to error
func SetSpanError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanOK sets the span status to OK
func SetSpanOK(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// AddSpanEvent adds an event to the span with optional attributes
func AddSpanEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on the span
func SetSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// Common attribute keys for consistent naming
var (
	// Task attributes
	AttrTaskID     = attribute.Key("task.id")
	AttrTaskStatus = attribute.Key("task.status")

	// Repository attributes
	AttrRepoFullName = attribute.Key("repo.full_name")
	AttrRepoOwner    = attribute.Key("repo.owner")
	AttrRepoName     = attribute.Key("repo.name")
	AttrRepoProvider = attribute.Key("repo.provider")
	AttrRepoRef      = attribute.Key("repo.ref")

	// Review attributes
	AttrReviewID     = attribute.Key("review.id")
	AttrReviewStatus = attribute.Key("review.status")
	AttrReviewAgent  = attribute.Key("review.agent")

	// Agent attributes
	AttrAgentName = attribute.Key("agent.name")
	AttrAgentType = attribute.Key("agent.type")

	// Result attributes
	AttrFindingsCount = attribute.Key("findings.count")
	AttrDurationMs    = attribute.Key("duration.ms")
)

// WithTaskAttributes returns span start options with task attributes
func WithTaskAttributes(taskID, repoFullName, ref string) trace.SpanStartOption {
	return trace.WithAttributes(
		AttrTaskID.String(taskID),
		AttrRepoFullName.String(repoFullName),
		AttrRepoRef.String(ref),
	)
}

// WithReviewAttributes returns span start options with review attributes
func WithReviewAttributes(reviewID string, agent string) trace.SpanStartOption {
	return trace.WithAttributes(
		AttrReviewID.String(reviewID),
		AttrReviewAgent.String(agent),
	)
}
