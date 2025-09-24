docker exec -it clickhouse-server ls /etc/clickhouse-server/users.d/

docker run -d --name clickhouse-server -v $(pwd)/clickhouse/otel-user.xml:/etc/clickhouse-server/users.d/otel-user.xml:ro -p 8123:8123 -p 9000:9000 -p 9363:9363 -v $(pwd)/clickhouse/prometheus.xml:/etc/clickhouse-server/config.d/prometheus.xml clickhouse/clickhouse-server:latest

docker exec -it clickhouse-server clickhouse-client --query "SHOW DATABASES;"

docker exec -it clickhouse-server clickhouse-client --query "SHOW TABLES FROM otel_metrics;"

docker exec -it clickhouse-server clickhouse-client --query "SHOW CREATE TABLE otel_metrics.otel_metrics_histogram;"

docker exec -it clickhouse-server clickhouse-client --query "SELECT * FROM otel_metrics.otel_metrics_histogram LIMIT 5;"


docker exec -it clickhouse-server clickhouse-client --query "SELECT * FROM otel_metrics.otel_metrics_histogram;"
