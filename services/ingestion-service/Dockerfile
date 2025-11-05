FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

RUN apt-get update && apt-get install -y protobuf-compiler && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Generate proto files inline - matching proto/logs.proto exactly
RUN mkdir -p /proto && \
    printf '%s\n' 'syntax = "proto3";' 'package logproto;' 'option go_package = "stackmonitor.com/ingestion-service/proto/logproto";' 'service LogIngestion { rpc StreamLogs(stream LogBatch) returns (stream Ack); }' 'message LogBatch { string agent_id = 1; int64 batch_id = 2; int64 timestamp_ms = 3; repeated LogEntry logs = 4; CompressionType compression = 5; bytes compressed_payload = 6; int32 original_size = 7; map<string, string> metadata = 8; }' 'message LogEntry { int64 timestamp_ns = 1; string level = 2; string message = 3; string source = 4; map<string, string> fields = 5; string agent_id = 6; }' 'message Ack { int64 batch_id = 1; AckStatus status = 2; string message = 3; int64 server_timestamp_ms = 4; }' 'enum AckStatus { SUCCESS = 0; RETRY = 1; DROP = 2; }' 'enum CompressionType { NONE = 0; ZSTD = 1; LOGLITE = 2; }' > /proto/logs.proto && \
    mkdir -p proto/logproto && \
    protoc --go_out=. --go_opt=module=stackmonitor.com/ingestion-service \
           --go-grpc_out=. --go-grpc_opt=module=stackmonitor.com/ingestion-service \
           --proto_path=/proto /proto/logs.proto && \
    echo 'module stackmonitor.com/logproto' > proto/logproto/go.mod

COPY . .

# Update go.mod to use the generated proto
RUN echo 'replace stackmonitor.com/ingestion-service/proto/logproto => ./proto/logproto' >> go.mod && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/ingestion-service .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/ingestion-service .

EXPOSE 50051

CMD ["./ingestion-service"]
