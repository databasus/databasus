package logging

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func newOpenTelemetryHandler(
	ctx context.Context,
	openTelemetry openTelemetrySettings,
	serviceVersion string,
) (slog.Handler, func(context.Context) error, error) {
	exporter, err := newOpenTelemetryExporter(ctx, openTelemetry)
	if err != nil {
		return nil, nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(newOpenTelemetryResource(serviceVersion)),
	)

	return otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(provider)), provider.Shutdown, nil
}

// newOpenTelemetryExporter passes the endpoint through WithEndpointURL so the path survives
// verbatim: a Collector listens on /v1/logs, VictoriaLogs on /insert/opentelemetry/v1/logs, and
// appending a fixed suffix would break one of them.
func newOpenTelemetryExporter(ctx context.Context, openTelemetry openTelemetrySettings) (sdklog.Exporter, error) {
	if openTelemetry.isGRPC {
		exporter, err := otlploggrpc.New(
			ctx,
			otlploggrpc.WithEndpointURL(openTelemetry.endpointURL),
			otlploggrpc.WithHeaders(openTelemetry.headers),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP/gRPC log exporter: %w", err)
		}

		return exporter, nil
	}

	exporter, err := otlploghttp.New(
		ctx,
		otlploghttp.WithEndpointURL(openTelemetry.endpointURL),
		otlploghttp.WithHeaders(openTelemetry.headers),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP/HTTP log exporter: %w", err)
	}

	return exporter, nil
}

func newOpenTelemetryResource(serviceVersion string) *resource.Resource {
	attributes := []attribute.KeyValue{semconv.ServiceName(serviceName)}
	if serviceVersion != "" {
		attributes = append(attributes, semconv.ServiceVersion(serviceVersion))
	}

	merged, err := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, attributes...))
	if err != nil {
		return resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	}

	return merged
}
