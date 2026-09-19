package upstream

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"torus-proxy/internal/observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestInstrumentedTransportSkipsMetricsWhenDisabled(t *testing.T) {
	observability.ResetForTesting()

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	tr := newInstrumentedTransport(base, false, "backend-a")

	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	observability.Handler().ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if strings.Contains(body, `torus_backend_requests_total{backend="backend-a"}`) {
		t.Fatalf("expected disabled transport to skip backend request metrics, got body:\n%s", body)
	}
}

// TestInstrumentedTransportCreatesClientSpan verifies that enabled upstream requests create a child client span and inject trace context.
func TestInstrumentedTransportCreatesClientSpan(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	parentTracer := provider.Tracer("test-parent")
	parentCtx, parentSpan := parentTracer.Start(context.Background(), "parent")
	defer parentSpan.End()

	var injectedTraceID trace.TraceID
	var injectedSpanID trace.SpanID

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		injectedCtx := propagation.TraceContext{}.Extract(
			context.Background(),
			propagation.HeaderCarrier(req.Header),
		)
		injectedSpanContext := trace.SpanContextFromContext(injectedCtx)

		injectedTraceID = injectedSpanContext.TraceID()
		injectedSpanID = injectedSpanContext.SpanID()

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"http://backend.example/api",
		nil,
	).WithContext(parentCtx)

	tr := newInstrumentedTransport(base, true, "backend-a")

	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended client span, got %d", len(ended))
	}

	clientSpan := ended[0]

	if clientSpan.SpanKind() != trace.SpanKindClient {
		t.Fatalf("expected CLIENT span, got %v", clientSpan.SpanKind())
	}

	if clientSpan.Parent().SpanID() != parentSpan.SpanContext().SpanID() {
		t.Fatalf(
			"expected client span parent %s, got %s",
			parentSpan.SpanContext().SpanID(),
			clientSpan.Parent().SpanID(),
		)
	}

	if clientSpan.Parent().TraceID() != parentSpan.SpanContext().TraceID() {
		t.Fatalf(
			"expected client span trace ID %s, got %s",
			parentSpan.SpanContext().TraceID(),
			clientSpan.Parent().TraceID(),
		)
	}

	if injectedTraceID != clientSpan.SpanContext().TraceID() {
		t.Fatalf(
			"expected injected trace ID %s, got %s",
			clientSpan.SpanContext().TraceID(),
			injectedTraceID,
		)
	}

	if injectedSpanID != clientSpan.SpanContext().SpanID() {
		t.Fatalf(
			"expected injected span ID %s, got %s",
			clientSpan.SpanContext().SpanID(),
			injectedSpanID,
		)
	}

	if clientSpan.Status().Code != codes.Unset {
		t.Fatalf("expected successful client span status to be unset, got %v", clientSpan.Status().Code)
	}
}

// TestInstrumentedTransportSkipsTracingWhenDisabled verifies that disabled observability creates no upstream span or trace propagation.
func TestInstrumentedTransportSkipsTracingWhenDisabled(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	parentTracer := provider.Tracer("test-parent")
	parentCtx, parentSpan := parentTracer.Start(context.Background(), "parent")
	defer parentSpan.End()

	req := httptest.NewRequest(
		http.MethodGet,
		"http://backend.example/api",
		nil,
	).WithContext(parentCtx)

	var traceparent string

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		traceparent = req.Header.Get("traceparent")

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	tr := newInstrumentedTransport(base, false, "backend-a")

	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	if len(recorder.Ended()) != 0 {
		t.Fatalf("expected no client spans, got %d", len(recorder.Ended()))
	}

	if traceparent != "" {
		t.Fatalf("expected no traceparent header, got %q", traceparent)
	}
}

// TestInstrumentedTransportMarksServerErrors verifies that upstream 5xx responses mark the client span as an error.
func TestInstrumentedTransportMarksServerErrors(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("bad gateway")),
			Header:     make(http.Header),
		}, nil
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"http://backend.example/api",
		nil,
	)

	tr := newInstrumentedTransport(base, true, "backend-a")

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, resp.StatusCode)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended client span, got %d", len(ended))
	}

	clientSpan := ended[0]

	if clientSpan.Status().Code != codes.Error {
		t.Fatalf("expected client span status ERROR, got %v", clientSpan.Status().Code)
	}
}

// TestInstrumentedTransportMarksTransportErrors verifies that upstream transport errors are recorded on the client span.
func TestInstrumentedTransportMarksTransportErrors(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, &testTransportError{message: "upstream failure"}
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"http://backend.example/api",
		nil,
	)

	tr := newInstrumentedTransport(base, true, "backend-a")

	if _, err := tr.RoundTrip(req); err == nil {
		t.Fatal("expected round trip to return transport error")
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended client span, got %d", len(ended))
	}

	clientSpan := ended[0]

	if clientSpan.Status().Code != codes.Error {
		t.Fatalf("expected client span status ERROR, got %v", clientSpan.Status().Code)
	}

	if len(clientSpan.Events()) == 0 {
		t.Fatal("expected client span to record the transport error")
	}
}

type testTransportError struct {
	message string
}

func (e *testTransportError) Error() string {
	return e.message
}
