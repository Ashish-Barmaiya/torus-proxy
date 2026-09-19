package observability

import (
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	httpServerTracerName = "torus-proxy/http-server"

	runtimeGenerationAttribute = "torus.runtime.generation"
	requestIDAttribute         = "torus.request.id"
)

// StartHTTPServerSpan extracts an incoming trace context and starts the
// server span for a Torus HTTP request.
//
// The returned request contains the new span context so downstream request
// processing can continue from the server span.
func StartHTTPServerSpan(
	r *http.Request,
	runtimeGeneration uint64,
) (*http.Request, trace.Span) {
	ctx := otel.GetTextMapPropagator().Extract(
		r.Context(),
		propagation.HeaderCarrier(r.Header),
	)

	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(r.Method),
		semconv.URLPathKey.String(r.URL.Path),
		semconv.URLSchemeKey.String(requestScheme(r)),
		attribute.Int64(
			runtimeGenerationAttribute,
			int64(runtimeGeneration),
		),
	}

	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		attrs = append(
			attrs,
			attribute.String(requestIDAttribute, requestID),
		)
	}

	ctx, span := otel.Tracer(httpServerTracerName).Start(
		ctx,
		r.Method,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attrs...),
	)

	return r.WithContext(ctx), span
}

// SetHTTPServerRoute records the matched Torus route and updates the span
// name using the HTTP server span naming convention.
//
// An empty route means the request was not matched and therefore must not
// populate http.route.
func SetHTTPServerRoute(
	span trace.Span,
	method string,
	route string,
) {
	if span == nil || route == "" {
		return
	}

	span.SetName(method + " " + route)
	span.SetAttributes(semconv.HTTPRouteKey.String(route))
}

// FinishHTTPServerSpan records the final HTTP status and ends the server span.
func FinishHTTPServerSpan(
	span trace.Span,
	status int,
) {
	if span == nil {
		return
	}

	span.SetAttributes(
		semconv.HTTPResponseStatusCodeKey.Int(status),
	)

	if status >= http.StatusInternalServerError &&
		status <= 599 {
		span.SetStatus(
			codes.Error,
			http.StatusText(status),
		)
		span.SetAttributes(
			semconv.ErrorTypeKey.String(strconv.Itoa(status)),
		)
	}

	span.End()
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}

	return "http"
}
