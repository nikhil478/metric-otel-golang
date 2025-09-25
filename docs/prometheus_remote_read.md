###### Prometheus Remote Read Request ######

message ReadRequest {
	repeated Query queries = 1;
	// accepted_response_types allows negotiating the content type of the response.
	//
	// Response types are taken from the list in the FIFO order. If no response type in `accepted_response_types` is
	// implemented by server, error is returned.
	// For request that do not contain `accepted_response_types` field the SAMPLES response type will be used.
	repeated ReadRequest.ResponseType accepted_response_types = 2;
}

message QueryResult {
	// Samples within a time series must be ordered by time.
	repeated TimeSeries timeseries = 1;
}

message TimeSeries {
	// For a timeseries to be valid, and for the samples and exemplars
	// to be ingested by the remote system properly, the labels field is required.
	repeated Label labels = 1 [(gogoproto.nullable) = false];
	repeated Sample samples = 2 [(gogoproto.nullable) = false];
	repeated Exemplar exemplars = 3 [(gogoproto.nullable) = false];
	repeated Histogram histograms = 4 [(gogoproto.nullable) = false];
}

message Label {
	string name = 1;
	string value = 2;
}

message Sample {
	double value = 1;
	// timestamp is in ms format, see model/timestamp/timestamp.go for
	// conversion from time.Time to Prometheus timestamp.
	int64 timestamp = 2;
}

message Exemplar {
	// Optional, can be empty.
	repeated Label labels = 1 [(gogoproto.nullable) = false];
	double value = 2;
	// timestamp is in ms format, see model/timestamp/timestamp.go for
	// conversion from time.Time to Prometheus timestamp.
	int64 timestamp = 3;
}

message Histogram {
	oneof count {
		uint64 count_int = 1;
		double count_float = 2;
	};
	double sum = 3;
	// The schema defines the bucket schema. Currently, valid numbers
	// are -4 <= n <= 8. They are all for base-2 bucket schemas, where 1
	// is a bucket boundary in each case, and then each power of two is
	// divided into 2^n logarithmic buckets. Or in other words, each
	// bucket boundary is the previous boundary times 2^(2^-n). In the
	// future, more bucket schemas may be added using numbers < -4 or >
	// 8.
	sint32 schema = 4;
	double zero_threshold = 5;
	oneof zero_count {
		uint64 zero_count_int = 6;
		double zero_count_float = 7;
	};
	// Negative Buckets.
	repeated BucketSpan negative_spans = 8 [(gogoproto.nullable) = false];
	// Use either "negative_deltas" or "negative_counts", the former for
	// regular histograms with integer counts, the latter for float
	// histograms.
	repeated sint64 negative_deltas = 9;
	repeated double negative_counts = 10;
	// Positive Buckets.
	repeated BucketSpan positive_spans = 11 [(gogoproto.nullable) = false];
	// Use either "positive_deltas" or "positive_counts", the former for
	// regular histograms with integer counts, the latter for float
	// histograms.
	repeated sint64 positive_deltas = 12;
	repeated double positive_counts = 13;
	Histogram.ResetHint reset_hint = 14;
	// timestamp is in ms format, see model/timestamp/timestamp.go for
	// conversion from time.Time to Prometheus timestamp.
	int64 timestamp = 15;
	// custom_values are not part of the specification, DO NOT use in remote write clients.
	// Used only for converting from OpenTelemetry to Prometheus internally.
	repeated double custom_values = 16;
}

enum ResponseType {
	// Server will return a single ReadResponse message with matched series that includes list of raw samples.
	// It's recommended to use streamed response types instead.
	//
	// Response headers:
	// Content-Type: "application/x-protobuf"
	// Content-Encoding: "snappy"
	SAMPLES = 0;
	// Server will stream a delimited ChunkedReadResponse message that
	// contains XOR or HISTOGRAM(!) encoded chunks for a single series.
	// Each message is following varint size and fixed size bigendian
	// uint32 for CRC32 Castagnoli checksum.
	//
	// Response headers:
	// Content-Type: "application/x-streamed-protobuf; proto=prometheus.ChunkedReadResponse"
	// Content-Encoding: ""
	STREAMED_XOR_CHUNKS = 1;
}


###### Prometheus Remote Read Response ######

message ReadResponse {
	// In same order as the request's queries.
	repeated QueryResult results = 1;
}

message QueryResult {
	// Samples within a time series must be ordered by time.
	repeated TimeSeries timeseries = 1;
}

message TimeSeries {
	// For a timeseries to be valid, and for the samples and exemplars
	// to be ingested by the remote system properly, the labels field is required.
	repeated Label labels = 1 [(gogoproto.nullable) = false];
	repeated Sample samples = 2 [(gogoproto.nullable) = false];
	repeated Exemplar exemplars = 3 [(gogoproto.nullable) = false];
	repeated Histogram histograms = 4 [(gogoproto.nullable) = false];
}