package observability

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type recordingExporter struct {
	spans []sdktrace.ReadOnlySpan
}

func (e *recordingExporter) ExportSpans(
	_ context.Context,
	spans []sdktrace.ReadOnlySpan,
) error {
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *recordingExporter) Shutdown(context.Context) error {
	return nil
}

func TestInitializeTracing(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()

	tracingMu.Lock()
	tracingInitialized = false
	tracingMu.Unlock()

	t.Cleanup(func() {
		tracingMu.Lock()
		tracingInitialized = false
		tracingMu.Unlock()

		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("OTEL_SERVICE_NAME", "")

	exporter := &recordingExporter{}

	shutdown, err := initializeTracing(
		context.Background(),
		func(context.Context) (sdktrace.SpanExporter, error) {
			return exporter, nil
		},
	)
	if err != nil {
		t.Fatalf("initialize tracing: %v", err)
	}

	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatalf(
			"global tracer provider is %T, want *trace.TracerProvider",
			otel.GetTracerProvider(),
		)
	}

	tracer := otel.Tracer("torus-proxy-test")

	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	if err := shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown tracing: %v", err)
	}

	if got := len(exporter.spans); got != 1 {
		t.Fatalf("exported spans = %d, want 1", got)
	}

	serviceNameFound := false

	for _, attr := range exporter.spans[0].Resource().Attributes() {
		if attr.Key != attribute.Key("service.name") {
			continue
		}

		serviceNameFound = attr.Value.AsString() == serviceName
		break
	}

	if !serviceNameFound {
		t.Fatalf(
			"exported span does not contain service.name=%q",
			serviceName,
		)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}

	if _, err := initializeTracing(
		context.Background(),
		func(context.Context) (sdktrace.SpanExporter, error) {
			t.Fatal("exporter factory called during reinitialization")
			return nil, nil
		},
	); err == nil {
		t.Fatal("expected reinitialization to fail")
	}

	fields := strings.Join(
		otel.GetTextMapPropagator().Fields(),
		",",
	)

	for _, field := range []string{
		"traceparent",
		"tracestate",
		"baggage",
	} {
		if !strings.Contains(fields, field) {
			t.Errorf(
				"global propagator fields %q do not contain %q",
				fields,
				field,
			)
		}
	}
}
