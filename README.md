# metric-otel-golang


docker run --rm -it -v $(pwd)/collector-config.yaml:/etc/otelcol/config.yaml -p 4317:4317 -p 4318:4318 otel/opentelemetry-collector:latest --config=/etc/otelcol/config.yaml


docker run --rm -it -v $(pwd)/collector-config.yaml:/etc/otelcol/config.yaml -p 4317:4317 -p 4318:4318 otel/opentelemetry-collector:latest --config=/etc/otelcol/config.yaml
