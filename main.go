package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"	
	apimetric "go.opentelemetry.io/otel/metric"	
)

var meter = otel.Meter("github.com/metric-otel-golang/metric")

func main() {

	res, err := newResource()
	if err != nil {
		panic(err)
	}

	meterProvider, err := newMeterProvider(res)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			log.Println(err)
		}
	}()

	otel.SetMeterProvider(meterProvider)

	histogram, err := meter.Float64Histogram(
		"task.duration",
		apimetric.WithDescription("The duration of task execution."),
		apimetric.WithUnit("s"),
		apimetric.WithExplicitBucketBoundaries(
        -5, -2, -1, 0, 1, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000,
    ),
	)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		time.Sleep(time.Second*3)
		duration := time.Since(start)
		histogram.Record(r.Context(), duration.Seconds())
		histogram.Record(r.Context(), -duration.Seconds())
	})

	http.ListenAndServe(":8080", nil)
}

func newResource() (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("test-service"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
}

func newMeterProvider(res *resource.Resource) (*metric.MeterProvider, error) {
	
	metricExporter, err := otlpmetricgrpc.New(context.Background(),
    otlpmetricgrpc.WithEndpoint("127.0.0.1:4317"),
    otlpmetricgrpc.WithInsecure(),
)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 3s for demonstrative purposes.
			metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}
