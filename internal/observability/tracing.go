package observability

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const serviceName = "torus-proxy"

var (
	tracingMu          sync.Mutex
	tracingInitialized bool
)

type spanExporterFactory func(context.Context) (sdktrace.SpanExporter, error)

// InitializeTracing creates and installs Torus's process-wide OpenTelemetry
// tracer provider.
//
// Tracing initialization is intentionally process-scoped. Runtime generations
// control whether instrumentation is emitted, but they do not own the tracer
// provider or exporter lifecycle.
func InitializeTracing(ctx context.Context) (func(context.Context) error, error) {
	return initializeTracing(ctx, func(ctx context.Context) (sdktrace.SpanExporter, error) {
		return otlptracehttp.New(ctx)
	})
}

func initializeTracing(
	ctx context.Context,
	exporterFactory spanExporterFactory,
) (func(context.Context) error, error) {
	tracingMu.Lock()
	defer tracingMu.Unlock()

	if tracingInitialized {
		return nil, errors.New("tracing has already been initialized")
	}

	exporter, err := exporterFactory(ctx)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil &&
		!errors.Is(err, resource.ErrPartialResource) &&
		!errors.Is(err, resource.ErrSchemaURLConflict) {
		_ = exporter.Shutdown(ctx)
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tracingInitialized = true

	var shutdownOnce sync.Once
	var shutdownErr error

	shutdown := func(shutdownCtx context.Context) error {
		shutdownOnce.Do(func() {
			shutdownErr = provider.Shutdown(shutdownCtx)
		})

		return shutdownErr
	}

	return shutdown, nil
}
