-- 5-minute aggregation table
CREATE TABLE IF NOT EXISTS otel_metrics.otel_metrics_5m
AS otel_metrics.otel_metrics_1m
ENGINE = MergeTree
PARTITION BY toDate(truncated_time)
ORDER BY (ServiceName, MetricName, truncated_time)
SETTINGS index_granularity = 8192;

-- 5-minute materialized view
CREATE MATERIALIZED VIEW IF NOT EXISTS otel_metrics.mv_otel_metrics_5m
TO otel_metrics.otel_metrics_5m
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
    toStartOfFiveMinute(truncated_time) AS truncated_time,

    avg(avg_value) AS avg_value,
    sum(total_count) AS total_count,
    sum(total_sum) AS total_sum,
    min(min_value) AS min_value,
    max(max_value) AS max_value,

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
FROM otel_metrics.otel_metrics_1m
GROUP BY ServiceName, MetricName, truncated_time;