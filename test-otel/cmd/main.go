package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
)

var db *sql.DB

func main() {

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	opts := &clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "otel_metrics",
			Username: "otel_user",
			Password: "otel_pass",
		},
	}
	var err error
	db = clickhouse.OpenDB(opts)
	if err = db.Ping(); err != nil {
		log.Fatalf("clickhouse ping: %v", err)
	}

	where := []string{"TimeUnix >= toDateTime64('2025-09-29 21:30:00',9) AND TimeUnix <= toDateTime64('2025-09-29 22:30:00',9)"}

	where = append(where, "MetricName = 'task_duration'")

	whereClause := strings.Join(where, " AND ")

	query := fmt.Sprintf(`SELECT
  MetricName,
  Attributes,
  toUnixTimestamp64Nano(TimeUnix) AS ts_ns,
  Sum,
  Count,
  BucketCounts,
  ExplicitBounds
FROM %s.%s
WHERE %s
LIMIT %d`, "otel_metrics", "otel_metrics_histogram", whereClause, 10)


	fmt.Printf("query %v \n", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {

		log.Printf("clickhouse query: %w", err)
		return
		// return nil, fmt.Errorf("clickhouse query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var metricName string
		var attributes map[string]string
		var tsNS int64
		var sum float64
		var count uint64
		var bucketCounts []uint64
		var explicitBounds []float64

		if err := rows.Scan(&metricName, &attributes, &tsNS, &sum, &count, &bucketCounts, &explicitBounds); err != nil {
			log.Printf("scan error: %v", err)
			continue
		}

		if len(bucketCounts) != len(explicitBounds) {
			continue
		}
		
		log.Printf("metricName: %v, attributes %v, tsNS %d, sum %f, count %d, bucketCounts %v, explicitBounds %v", metricName, attributes, tsNS, sum, count, bucketCounts, explicitBounds)
	}
}
