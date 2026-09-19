package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// TestTracing_EndToEnd verifies that a real proxied request produces a server span,
// a child upstream client span, and W3C trace-context propagation to the backend.
func TestTracing_EndToEnd(t *testing.T) {
	resetTestState(t)

	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	backendSpanContextCh := make(chan trace.SpanContext, 1)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagation.TraceContext{}.Extract(
			r.Context(),
			propagation.HeaderCarrier(r.Header),
		)

		backendSpanContextCh <- trace.SpanContextFromContext(ctx)

		_, _ = io.WriteString(w, "backend")
	}))

	defer backend.Close()

	configPath := newTempConfig(t)
	writeConfig(
		t,
		configPath,
		"127.0.0.1:0",
		backend.URL,
	)

	_, server, _, baseURL := startProxy(t, configPath)
	defer func() {
		if err := server.Shutdown(5 * time.Second); err != nil {
			t.Errorf("shutdown proxy: %v", err)
		}
	}()

	parentTraceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929f3e0e4736")
	if err != nil {
		t.Fatalf("parse parent trace ID: %v", err)
	}

	parentSpanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("parse parent span ID: %v", err)
	}

	parentSpanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    parentTraceID,
		SpanID:     parentSpanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	parentCtx := trace.ContextWithSpanContext(
		context.Background(),
		parentSpanContext,
	)

	req, err := http.NewRequestWithContext(
		parentCtx,
		http.MethodGet,
		baseURL+"/api",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	otel.GetTextMapPropagator().Inject(
		parentCtx,
		propagation.HeaderCarrier(req.Header),
	)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if string(body) != "backend" {
		t.Fatalf("expected backend response, got %q", string(body))
	}

	var backendSpanContext trace.SpanContext

	select {
	case backendSpanContext = <-backendSpanContextCh:
		if !backendSpanContext.IsValid() {
			t.Fatal("backend received an invalid trace context")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for backend trace context")
	}

	spans := recorder.Ended()

	if len(spans) != 2 {
		t.Fatalf("recorded spans = %d, want 2", len(spans))
	}

	var (
		serverSpan sdktrace.ReadOnlySpan
		clientSpan sdktrace.ReadOnlySpan
	)

	for _, span := range spans {
		switch span.SpanKind() {
		case trace.SpanKindServer:
			serverSpan = span
		case trace.SpanKindClient:
			clientSpan = span
		}
	}

	if serverSpan == nil {
		t.Fatal("expected HTTP server span")
	}

	if clientSpan == nil {
		t.Fatal("expected HTTP client span")
	}

	if serverSpan.Parent().SpanID() != parentSpanID {
		t.Fatalf(
			"server span parent = %s, want %s",
			serverSpan.Parent().SpanID(),
			parentSpanID,
		)
	}

	if serverSpan.SpanContext().TraceID() != parentTraceID {
		t.Fatalf(
			"server trace ID = %s, want %s",
			serverSpan.SpanContext().TraceID(),
			parentTraceID,
		)
	}

	if clientSpan.Parent().SpanID() != serverSpan.SpanContext().SpanID() {
		t.Fatalf(
			"client span parent = %s, want %s",
			clientSpan.Parent().SpanID(),
			serverSpan.SpanContext().SpanID(),
		)
	}

	if clientSpan.SpanContext().TraceID() != serverSpan.SpanContext().TraceID() {
		t.Fatalf(
			"client trace ID = %s, want server trace ID %s",
			clientSpan.SpanContext().TraceID(),
			serverSpan.SpanContext().TraceID(),
		)
	}

	if backendSpanContext.TraceID() != clientSpan.SpanContext().TraceID() {
		t.Fatalf(
			"backend trace ID = %s, want client trace ID %s",
			backendSpanContext.TraceID(),
			clientSpan.SpanContext().TraceID(),
		)
	}

	if backendSpanContext.SpanID() != clientSpan.SpanContext().SpanID() {
		t.Fatalf(
			"backend span ID = %s, want client span ID %s",
			backendSpanContext.SpanID(),
			clientSpan.SpanContext().SpanID(),
		)
	}
}
