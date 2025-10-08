package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	prompb "github.com/prometheus/prometheus/prompb"
)

var (
	chAddr       = envOr("CLICKHOUSE_ADDR", "clickhouse-server:9000")
	chDatabase   = envOr("CLICKHOUSE_DB", "otel_metrics")
	chUser       = envOr("CLICKHOUSE_USER", "otel_user")
	chPass       = envOr("CLICKHOUSE_PASS", "otel_pass")
	chTable      = envOr("CLICKHOUSE_TABLE", "otel_metrics_sum")
	listenAddr   = envOr("PROXY_LISTEN", ":9364")
	queryTimeout = envDurationOr("QUERY_TIMEOUT", 30*time.Second)
	maxRows      = envIntOr("MAX_ROWS", 20000)
)

// retention windows (seconds)
const (
	rawRetentionSec  = 3 * 3600        // 3 hours
	oneMinRetention  = 15 * 24 * 3600  // 15 days
	fiveMinRetention = 63 * 24 * 3600  // 63 days
	oneHourRetention = 455 * 24 * 3600 // 455 days
)

// table names (fully qualified)
var (
	rawTable   = "otel_metrics.otel_metrics_sum"
	oneMinTbl  = "otel_metrics.otel_metrics_sum_1m"
	fiveMinTbl = "otel_metrics.otel_metrics_sum_5m"
	oneHourTbl = "otel_metrics.otel_metrics_sum_1h"
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envDurationOr(k string, d time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if t, err := time.ParseDuration(v); err == nil {
			return t
		}
	}
	return d
}
func envIntOr(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}

var db *sql.DB

func main() {
	flag.Parse()

	opts := &clickhouse.Options{
		Addr: []string{chAddr},
		Auth: clickhouse.Auth{
			Database: chDatabase,
			Username: chUser,
			Password: chPass,
		},
	}
	var err error
	db = clickhouse.OpenDB(opts)
	if err = db.Ping(); err != nil {
		log.Fatalf("clickhouse ping: %v", err)
	}

	http.HandleFunc("/read", handleRemoteRead)
	log.Printf("listening on %s, ClickHouse %s.%s", listenAddr, chDatabase, chTable)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func handleRemoteRead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	// read body
	compBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed read body", http.StatusBadRequest)
		return
	}
	reqBuf, err := snappy.Decode(nil, compBody)
	if err != nil {
		http.Error(w, "failed snappy decode", http.StatusBadRequest)
		return
	}
	var rr prompb.ReadRequest
	if err := proto.Unmarshal(reqBuf, &rr); err != nil {
		http.Error(w, "failed proto unmarshal", http.StatusBadRequest)
		return
	}

	resp := &prompb.ReadResponse{}
	for _, q := range rr.Queries {
		ts, err := processQuerySum(ctx, q)
		if err != nil {
			log.Printf("processQuery error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		qr := &prompb.QueryResult{
			Timeseries: ts,
		}
		resp.Results = append(resp.Results, qr)
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, "failed proto marshal", http.StatusInternalServerError)
		return
	}
	enc := snappy.Encode(nil, out)
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Content-Encoding", "snappy")
	_, _ = w.Write(enc)
}

func ProcessQuery(ctx context.Context, q *prompb.Query) ([]*prompb.TimeSeries, error) {
	var metricNameEq string
	labelEq := map[string]string{}
	for _, m := range q.Matchers {
		if m.Type == prompb.LabelMatcher_EQ {
			if m.Name == "__name__" {
				metricNameEq = m.Value
			} else {
				labelEq[m.Name] = m.Value
			}
		}
	}

	// Convert start/end to seconds
	startMs := q.StartTimestampMs
	endMs := q.EndTimestampMs
	if endMs == 0 {
		endMs = time.Now().UnixNano() / 1e6
	}
	startSec := float64(startMs) / 1000.0
	endSec := float64(endMs) / 1000.0

	// Determine wantType and baseMetric
	wantType := ""
	baseMetric := ""
	if metricNameEq != "" {
		switch {
		case strings.HasSuffix(metricNameEq, "_bucket"):
			wantType = "bucket"
			baseMetric = strings.TrimSuffix(metricNameEq, "_bucket")
		case strings.HasSuffix(metricNameEq, "_sum"):
			wantType = "sum"
			baseMetric = strings.TrimSuffix(metricNameEq, "_sum")
		case strings.HasSuffix(metricNameEq, "_count"):
			wantType = "count"
			baseMetric = strings.TrimSuffix(metricNameEq, "_count")
		default:
			baseMetric = metricNameEq
		}
	}

	// Build WHERE clause and arguments
	where := []string{"TimeUnix >= toDateTime64(?,9) AND TimeUnix <= toDateTime64(?,9)"}
	args := []interface{}{startSec, endSec}

	if baseMetric != "" {
		where = append(where, "MetricName = ?")
		args = append(args, baseMetric)
	}

	leFilter := ""
	if v, ok := labelEq["le"]; ok {
		leFilter = v
		delete(labelEq, "le")
	}
	for k, v := range labelEq {
		ek := strings.ReplaceAll(k, "'", "\\'")
		where = append(where, fmt.Sprintf("Attributes['%s'] = ?", ek))
		args = append(args, v)
	}

	whereClause := strings.Join(where, " AND ")

	query := fmt.Sprintf(`
SELECT
  MetricName,
  Attributes,
  toUnixTimestamp64Nano(TimeUnix) AS ts_ns,
  Sum,
  Count,
  BucketCounts,
  ExplicitBounds
FROM %s.%s
WHERE %s
LIMIT %d
`, chDatabase, chTable, whereClause, maxRows)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse query: %w", err)
	}
	defer rows.Close()

	var out []*prompb.TimeSeries

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

		baseLabels := []prompb.Label{}
		for k, v := range attributes {
			baseLabels = append(baseLabels, prompb.Label{Name: k, Value: v})
		}

		// Buckets (including +Inf)
		cum := uint64(0)
		for i := 0; i < len(bucketCounts); i++ {
			cum += bucketCounts[i]

			leStr := "+Inf"
			if i < len(explicitBounds) {
				leStr = strconv.FormatFloat(explicitBounds[i], 'g', -1, 64)
			}

			if wantType == "" || wantType == "bucket" {
				if leFilter != "" && leFilter != leStr {
					continue
				}
				labels := append([]prompb.Label{
					{Name: "__name__", Value: metricName + "_bucket"},
					{Name: "le", Value: leStr},
				}, baseLabels...)

				out = append(out, &prompb.TimeSeries{
					Labels:  labels,
					Samples: []prompb.Sample{{Timestamp: tsNS / 1e6, Value: float64(cum)}},
				})
			}
		}

		// Sum
		if wantType == "" || wantType == "sum" {
			labels := append([]prompb.Label{{Name: "__name__", Value: metricName + "_sum"}}, baseLabels...)
			out = append(out, &prompb.TimeSeries{
				Labels:  labels,
				Samples: []prompb.Sample{{Timestamp: tsNS / 1e6, Value: sum}},
			})
		}

		// Count
		if wantType == "" || wantType == "count" {
			labels := append([]prompb.Label{{Name: "__name__", Value: metricName + "_count"}}, baseLabels...)
			out = append(out, &prompb.TimeSeries{
				Labels:  labels,
				Samples: []prompb.Sample{{Timestamp: tsNS / 1e6, Value: float64(count)}},
			})
		}
	}

	return out, nil
}

func processQuerySum(ctx context.Context, q *prompb.Query) ([]*prompb.TimeSeries, error) {
	log.Printf("Prometheus query looks like: %v", q)

	metricNameEq := ""
	labelEq := map[string]string{}

	for _, m := range q.Matchers {
		if m.Type == prompb.LabelMatcher_EQ {
			if m.Name == "__name__" {
				metricNameEq = m.Value
			} else {
				labelEq[m.Name] = m.Value
			}
		}
	}

	startMs := q.StartTimestampMs
	endMs := q.EndTimestampMs
	if endMs == 0 {
		endMs = time.Now().UnixNano() / 1e6
	}
	startSec := float64(startMs) / 1000.0
	endSec := float64(endMs) / 1000.0

	nowSec := float64(time.Now().Unix())

	rawCut := nowSec - rawRetentionSec
	oneMinCut := nowSec - oneMinRetention
	fiveMinCut := nowSec - fiveMinRetention

	type segment struct {
		table string
		from  float64
		to    float64
	}

	segments := []segment{}
	addIf := func(tbl string, s, e float64) {
		if s < e && e > startSec && s < endSec {
			s2 := math.Max(s, startSec)
			e2 := math.Min(e, endSec)
			if s2 < e2 {
				segments = append(segments, segment{table: tbl, from: s2, to: e2})
			}
		}
	}

	addIf(rawTable, math.Max(startSec, rawCut), endSec)                           
	addIf(oneMinTbl, math.Max(startSec, oneMinCut), math.Min(endSec, rawCut))      
	addIf(fiveMinTbl, math.Max(startSec, fiveMinCut), math.Min(endSec, oneMinCut))
	addIf(oneHourTbl, startSec, math.Min(endSec, fiveMinCut))                      

	if len(segments) == 0 {
		return nil, nil
	}

	labelWhereParts := []string{}
	labelArgs := []interface{}{}
	if metricNameEq != "" {
		labelWhereParts = append(labelWhereParts, "MetricName = ?")
		labelArgs = append(labelArgs, metricNameEq)
	}
	for k, v := range labelEq {
		ek := strings.ReplaceAll(k, "'", "\\'")
		labelWhereParts = append(labelWhereParts, fmt.Sprintf("Attributes['%s'] = ?", ek))
		labelArgs = append(labelArgs, v)
	}
	labelWhere := ""
	if len(labelWhereParts) > 0 {
		labelWhere = " AND " + strings.Join(labelWhereParts, " AND ")
	}

	selects := make([]string, 0, len(segments))
	args := []interface{}{}

	for _, s := range segments {
		valueExpr := "SumValue"
		if s.table == rawTable {
			valueExpr = "Value AS SumValue"
		}

		segWhere := "TimeUnix >= toDateTime64(?,9) AND TimeUnix < toDateTime64(?,9)"
		fullWhere := segWhere + labelWhere

		qStr := fmt.Sprintf(`
SELECT
  MetricName,
  Attributes,
  toUnixTimestamp64(TimeUnix) AS ts_ns,
  %s
FROM %s
WHERE %s
`, valueExpr, s.table, fullWhere)

		selects = append(selects, qStr)
		args = append(args, s.from, s.to)
		args = append(args, labelArgs...)
	}

	unionQuery := strings.Join(selects, "\nUNION ALL\n") + fmt.Sprintf("\nORDER BY ts_ns\nLIMIT %d", maxRows)

	rows, err := db.QueryContext(ctx, unionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("ClickHouse query error: %w", err)
	}
	defer rows.Close()

	var out []*prompb.TimeSeries
	for rows.Next() {
		var metricName string
		var attributes map[string]string
		var tsNS int64
		var sumValue float64

		if err := rows.Scan(&metricName, &attributes, &tsNS, &sumValue); err != nil {
			log.Printf("row scan error: %v", err)
			continue
		}

		labels := []prompb.Label{{Name: "__name__", Value: metricName}}
		for k, v := range attributes {
			labels = append(labels, prompb.Label{Name: k, Value: v})
		}

		ts := &prompb.TimeSeries{
			Labels:  labels,
			Samples: []prompb.Sample{{Timestamp: tsNS / 1e6, Value: sumValue}},
		}

		out = append(out, ts)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
