FROM golang:1.21 AS builder

WORKDIR /app

# Generate proto file directly in container
RUN mkdir -p /proto && \
    printf 'syntax = "proto3";\n\npackage logproto;\n\noption go_package = "stackmonitor.com/logproto";\n\nservice LogIngestion {\n  rpc StreamLogs(stream LogBatch) returns (stream Ack);\n}\n\nmessage LogBatch {\n  string agent_id = 1;\n  int64 batch_id = 2;\n  int64 timestamp_ms = 3;\n  repeated LogEntry logs = 4;\n  CompressionType compression = 5;\n  bytes compressed_payload = 6;\n  int32 original_size = 7;\n  map<string, string> metadata = 8;\n}\n\nmessage LogEntry {\n  int64 timestamp_ns = 1;\n  string level = 2;\n  string message = 3;\n  string source = 4;\n  map<string, string> fields = 5;\n  string agent_id = 6;\n}\n\nmessage Ack {\n  int64 batch_id = 1;\n  AckStatus status = 2;\n  string message = 3;\n  int64 server_timestamp_ms = 4;\n}\n\nenum AckStatus {\n  SUCCESS = 0;\n  RETRY = 1;\n  DROP = 2;\n}\n\nenum CompressionType {\n  NONE = 0;\n  ZSTD = 1;\n  LOGLITE = 2;\n}\n' > /proto/logs.proto

COPY go.mod ./

COPY . .

RUN apt-get update && apt-get install -y protobuf-compiler && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

RUN mkdir -p proto && \
    protoc --go_out=./proto --go_opt=paths=source_relative \
           --go-grpc_out=./proto --go-grpc_opt=paths=source_relative \
           --proto_path=/proto /proto/logs.proto && \
    echo 'module stackmonitor.com/logproto\n\ngo 1.21' > proto/go.mod

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ingestion-service .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/ingestion-service .

EXPOSE 50051

CMD ["./ingestion-service"]
