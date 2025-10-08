CREATE DATABASE IF NOT EXISTS otel_metrics;

CREATE TABLE IF NOT EXISTS otel_metrics.otel_metrics_sum_1m
(
    MetricName String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    SumValue Float64 CODEC(ZSTD(1))
)
ENGINE = SummingMergeTree()
PARTITION BY toDate(TimeUnix)
ORDER BY (MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS otel_metrics.otel_metrics_sum_5m
(
    MetricName String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    SumValue Float64 CODEC(ZSTD(1))
)
ENGINE = SummingMergeTree()
PARTITION BY toDate(TimeUnix)
ORDER BY (MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS otel_metrics.otel_metrics_sum_1h
(
    MetricName String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    SumValue Float64 CODEC(ZSTD(1))
)
ENGINE = SummingMergeTree()
PARTITION BY toDate(TimeUnix)
ORDER BY (MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity = 8192;


CREATE MATERIALIZED VIEW IF NOT EXISTS otel_metrics.mv_sum_raw_to_1m
TO otel_metrics.otel_metrics_sum_1m
AS
SELECT
  MetricName,
  Attributes,
  toDateTime64(intDiv(toUnixTimestamp64(TimeUnix), 60) * 60, 9) AS TimeUnix,
  sum(Value) AS SumValue
FROM otel_metrics.otel_metrics_sum
GROUP BY
  MetricName,
  Attributes,
  TimeUnix
;

CREATE MATERIALIZED VIEW IF NOT EXISTS otel_metrics.mv_sum_1m_to_5m
TO otel_metrics.otel_metrics_sum_5m
AS
SELECT
  MetricName,
  Attributes,
  toDateTime64(intDiv(toUnixTimestamp64(TimeUnix), 300) * 300, 9) AS TimeUnix,
  sum(SumValue) AS SumValue
FROM otel_metrics.otel_metrics_sum_1m
GROUP BY
  MetricName,
  Attributes,
  TimeUnix
;

CREATE MATERIALIZED VIEW IF NOT EXISTS otel_metrics.mv_sum_5m_to_1h
TO otel_metrics.otel_metrics_sum_1h
AS
SELECT
  MetricName,
  Attributes,
  toDateTime64(intDiv(toUnixTimestamp64(TimeUnix), 3600) * 3600, 9) AS TimeUnix,
  sum(SumValue) AS SumValue
FROM otel_metrics.otel_metrics_sum_5m
GROUP BY
  MetricName,
  Attributes,
  TimeUnix
;
