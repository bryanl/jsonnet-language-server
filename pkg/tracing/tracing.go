package tracing

import (
	"context"

	opentracing "github.com/opentracing/opentracing-go"
)

// ChildSpan creates a child span given a context.
func ChildSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {
	parent := opentracing.SpanFromContext(ctx)
	span := parent.Tracer().StartSpan(
		name,
		opentracing.ChildOf(parent.Context()),
	)

	childCtx := opentracing.ContextWithSpan(ctx, span)
	return span, childCtx
}
