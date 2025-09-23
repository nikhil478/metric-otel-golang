

docker run --rm -it --network otel-network -v $(pwd)/otel-collector/collector-config.yaml:/etc/otelcol/config.yaml -p 4317:4317 -p 4318:4318 otel/opentelemetry-collector-contrib:latest --config=/etc/otelcol/config.yaml