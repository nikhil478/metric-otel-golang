docker build -t ch-otel-prom-proxy:latest .


docker run --rm -p 9364:9364 \
  --name ch-otel-prom-proxy \
  --network otel-network \
  -e CLICKHOUSE_HOST=clickhouse-server \
  -e CLICKHOUSE_PORT=9000 \
  ch-otel-prom-proxy:latest
