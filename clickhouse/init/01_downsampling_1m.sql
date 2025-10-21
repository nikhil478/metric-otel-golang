-- 1-minute aggregation table
CREATE TABLE IF NOT EXISTS otel_metrics.otel_metrics_1m
(
    ServiceName LowCardinality(String),
    MetricName String,
    ResourceSchemaUrl String,
    ScopeName String,
    ScopeVersion String,
    ScopeSchemaUrl String,
    ResourceAttributes Array(Map(LowCardinality(String), String)),
    ScopeAttributes Array(Map(LowCardinality(String), String)),
    Attributes Array(Map(LowCardinality(String), String)),
    StartTimeUnix DateTime64(9),
    truncated_time DateTime64(0),
    
    avg_value Float64,
    total_count UInt64,
    total_sum Float64,
    min_value Float64,
    max_value Float64,

    BucketCounts Array(UInt64),
    ExplicitBounds Array(Float64),

    Scale Float32,
    ZeroCount UInt64,
    PositiveOffset Int32,
    PositiveBucketCounts Array(UInt64),
    NegativeOffset Int32,
    NegativeBucketCounts Array(UInt64),

    `ValueAtQuantiles.Quantile` Array(Float64),
    `ValueAtQuantiles.Value` Array(Float64),

    `Exemplars.FilteredAttributes` Array(Map(LowCardinality(String), String)),
    `Exemplars.TimeUnix` Array(DateTime64(9)),
    `Exemplars.Value` Array(Float64),
    `Exemplars.SpanId` Array(String),
    `Exemplars.TraceId` Array(String),

    Flags UInt32,
    AggregationTemporality Int32,
    IsMonotonic UInt8
)
ENGINE = MergeTree
PARTITION BY toDate(truncated_time)
ORDER BY (ServiceName, MetricName, truncated_time)
SETTINGS index_granularity = 8192;


-- 1-minute materialized view
CREATE MATERIALIZED VIEW IF NOT EXISTS otel_metrics.mv_otel_metrics_1m
TO otel_metrics.otel_metrics_1m
AS
SELECT
    ServiceName,
    MetricName,
    any(ResourceSchemaUrl) AS ResourceSchemaUrl,
    any(ScopeName) AS ScopeName,
    any(ScopeVersion) AS ScopeVersion,
    any(ScopeSchemaUrl) AS ScopeSchemaUrl,
    arrayFlatten(groupArray(ResourceAttributes)) AS ResourceAttributes,
    arrayFlatten(groupArray(ScopeAttributes)) AS ScopeAttributes,
    arrayFlatten(groupArray(Attributes)) AS Attributes,
    min(StartTimeUnix) AS StartTimeUnix,
    toStartOfMinute(TimeUnix) AS truncated_time,

    avg(Value) AS avg_value,
    sum(Count) AS total_count,
    sum(Sum) AS total_sum,
    min(Min) AS min_value,
    max(Max) AS max_value,

    arrayFlatten(groupArray(BucketCounts)) AS BucketCounts,
    arrayFlatten(groupArray(ExplicitBounds)) AS ExplicitBounds,

    avg(Scale) AS Scale,
    sum(ZeroCount) AS ZeroCount,
    avg(PositiveOffset) AS PositiveOffset,
    arrayFlatten(groupArray(PositiveBucketCounts)) AS PositiveBucketCounts,
    avg(NegativeOffset) AS NegativeOffset,
    arrayFlatten(groupArray(NegativeBucketCounts)) AS NegativeBucketCounts,

    arrayFlatten(groupArray(`ValueAtQuantiles.Quantile`)) AS `ValueAtQuantiles.Quantile`,
    arrayFlatten(groupArray(`ValueAtQuantiles.Value`)) AS `ValueAtQuantiles.Value`,

    arrayFlatten(groupArray(`Exemplars.FilteredAttributes`)) AS `Exemplars.FilteredAttributes`,
    arrayFlatten(groupArray(`Exemplars.TimeUnix`)) AS `Exemplars.TimeUnix`,
    arrayFlatten(groupArray(`Exemplars.Value`)) AS `Exemplars.Value`,
    arrayFlatten(groupArray(`Exemplars.SpanId`)) AS `Exemplars.SpanId`,
    arrayFlatten(groupArray(`Exemplars.TraceId`)) AS `Exemplars.TraceId`,

    any(Flags) AS Flags,
    any(AggregationTemporality) AS AggregationTemporality,
    max(IsMonotonic) AS IsMonotonic
FROM otel_metrics.otel_metrics_all
GROUP BY ServiceName, MetricName, truncated_time;