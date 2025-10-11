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

	gauge, err := meter.Float64ObservableGauge(
		"random_value_gauge",
		apimetric.WithDescription("A random value between -20 and +20 recorded on each request"),
		apimetric.WithUnit("1"),
	)

	if err != nil {
		panic(err)
	}

	var lastValue float64

	_, err = meter.RegisterCallback(
		func(ctx context.Context, o apimetric.Observer) error {
			o.ObserveFloat64(gauge, lastValue)
			return nil
		},
		gauge,
	)

	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lastValue = float64(rand.Intn(41) - 20)

		fmt.Fprintf(w, "Recorded gauge value: %.2f\nProcessing time: %s\n",
			lastValue, time.Since(start))

	})

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
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(2*time.Second))),
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
