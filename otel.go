package main

import (
	"context"
	"errors"
	"log"
	"time"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up resource.
	res, err := newResource(serviceName, serviceVersion)
	if err != nil {
		handleErr(err)
		return
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	exporter, _ := newExporter(ctx)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider(res, exporter)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider(res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return
}

func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		))
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	// collectorEndpoint := "http://127.0.0.1:56711" // Sending traces and spans to local collector
	// collectorEndpoint := os.Getenv("OTEL_COLLECTOR_ENDPOINT")
	// traceExporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(collectorEndpoint))
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		log.Printf("Error sending traces %s", err)
		panic(err)

	}
	return traceExporter, nil
}

func newTraceProvider(res *resource.Resource, exporter *otlptrace.Exporter) (*trace.TracerProvider, error) {
	// trace exporter for stdout
	// traceExporter, err := stdouttrace.New(
	// 	stdouttrace.WithPrettyPrint())s
	// if err != nil {
	// 	return nil, err
	// }

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
	)
	log.Print("Sent to otel collector")
	return traceProvider, nil
}

func newMeterProvider(res *resource.Resource) (*metric.MeterProvider, error) {
	// metricExporter, err := stdoutmetric.New()
	// if err != nil {
	// 	return nil, err
	// }

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		// metric.WithReader(
		// 	metric.NewPeriodicReader(metricExporter,
		// 	// Default is 1m. Set to 3s for demonstrative purposes.
		// 	metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}
