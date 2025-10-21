CREATE DATABASE IF NOT EXISTS otel_metrics;

CREATE TABLE IF NOT EXISTS otel_metrics.otel_metrics_all
(
    `ResourceAttributes` Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    `ResourceSchemaUrl` String CODEC(ZSTD(1)),
    `ScopeName` String CODEC(ZSTD(1)),
    `ScopeVersion` String CODEC(ZSTD(1)),
    `ScopeAttributes` Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    `ScopeDroppedAttrCount` UInt32 CODEC(ZSTD(1)),
    `ScopeSchemaUrl` String CODEC(ZSTD(1)),
    `ServiceName` LowCardinality(String) CODEC(ZSTD(1)),
    `MetricName` String CODEC(ZSTD(1)),
    `MetricDescription` String CODEC(ZSTD(1)),
    `MetricUnit` String CODEC(ZSTD(1)),
    `Attributes` Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    `StartTimeUnix` DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    `TimeUnix` DateTime64(9) CODEC(Delta(8), ZSTD(1)),

    `Value` Nullable(Float64) CODEC(ZSTD(1)),
    `Count` Nullable(UInt64) CODEC(Delta(8), ZSTD(1)),
    `Sum` Nullable(Float64) CODEC(ZSTD(1)),
    `Min` Nullable(Float64) CODEC(ZSTD(1)),
    `Max` Nullable(Float64) CODEC(ZSTD(1)),

    `BucketCounts` Array(UInt64) CODEC(ZSTD(1)),
    `ExplicitBounds` Array(Float64) CODEC(ZSTD(1)),

    `Scale` Nullable(Int32) CODEC(ZSTD(1)),
    `ZeroCount` Nullable(UInt64) CODEC(ZSTD(1)),
    `PositiveOffset` Nullable(Int32) CODEC(ZSTD(1)),
    `PositiveBucketCounts` Array(UInt64) CODEC(ZSTD(1)),
    `NegativeOffset` Nullable(Int32) CODEC(ZSTD(1)),
    `NegativeBucketCounts` Array(UInt64) CODEC(ZSTD(1)),

    `ValueAtQuantiles.Quantile` Array(Float64) CODEC(ZSTD(1)),
    `ValueAtQuantiles.Value` Array(Float64) CODEC(ZSTD(1)),

    `IsMonotonic` Nullable(Bool) CODEC(Delta(1), ZSTD(1)),

    `Exemplars.FilteredAttributes` Array(Map(LowCardinality(String), String)) CODEC(ZSTD(1)),
    `Exemplars.TimeUnix` Array(DateTime64(9)) CODEC(ZSTD(1)),
    `Exemplars.Value` Array(Float64) CODEC(ZSTD(1)),
    `Exemplars.SpanId` Array(String) CODEC(ZSTD(1)),
    `Exemplars.TraceId` Array(String) CODEC(ZSTD(1)),

    `Flags` UInt32 CODEC(ZSTD(1)),
    `AggregationTemporality` Nullable(Int32) CODEC(ZSTD(1))
)
ENGINE = MergeTree
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity = 8192;