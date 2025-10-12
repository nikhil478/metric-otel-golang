package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	apimetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

var meter = otel.Meter("github.com/metric-otel-golang/metric")

func main() {
	res, err := newResource()
	if err != nil {
		panic(err)
	}

	meterProvider, _ := newMeterProvider(res)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			log.Println("Error shutting down meter provider:", err)
		}
	}()

	otel.SetMeterProvider(meterProvider)

	histogram, err := meter.Float64Histogram(
		"request_duration_ms",
		apimetric.WithDescription("Exponential histogram of request durations in ms"),
		apimetric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		for i := 0; i < 10; i++ {
			duration := float64(rand.Intn(1000) + 1)
			log.Println("Recording request_duration_ms:", duration)
			histogram.Record(context.Background(), duration)
		}

		fmt.Fprintf(w, "Recorded 10 random request durations\nProcessing time: %s\n", time.Since(start))
	})

	fmt.Println("Listening on :8080 ...")
	http.ListenAndServe(":8080", nil)
}

func newMeterProvider(res *resource.Resource) (*metric.MeterProvider, *metric.PeriodicReader) {
	metricExporter, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithEndpoint("127.0.0.1:4317"),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	reader := metric.NewPeriodicReader(
		metricExporter,
		metric.WithInterval(2*time.Second),
	)

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
		metric.WithView(metric.NewView(
			metric.Instrument{Name: "request_duration_ms", Kind: metric.InstrumentKindHistogram},
			metric.Stream{Aggregation: metric.AggregationBase2ExponentialHistogram{
				MaxSize:  160,   // must be > 0
				MaxScale: 8,     
			}},
		)),
	)

	return meterProvider, reader
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
