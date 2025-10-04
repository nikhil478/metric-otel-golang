package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
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

	requestCounter, err := meter.Float64Counter(
		"http_requests_total",
		apimetric.WithDescription("Total number of HTTP requests processed."),
		apimetric.WithUnit("1"),
	)

	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		time.Sleep(2 * time.Second)
		duration := time.Since(start)

		requestCounter.Add(r.Context(), 1)

		durationSum, _ := meter.Float64Counter(
			"http_request_duration_seconds_sum",
			apimetric.WithDescription("Total accumulated request duration."),
			apimetric.WithUnit("s"),
		)

		durationSum.Add(r.Context(), duration.Seconds())

		w.Write([]byte("Request processed\n"))
	})

	log.Println("Listening on :8080 ...")
	http.ListenAndServe(":8080", nil)
}

func newMeterProvider(res *resource.Resource) (*metric.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithEndpoint("127.0.0.1:4317"),
		otlpmetricgrpc.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(3*time.Second))),
	)

	return meterProvider, nil
}

func newResource() (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(), resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("test-service"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
}
