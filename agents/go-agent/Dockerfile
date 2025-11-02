FROM golang:1.21 AS builder

WORKDIR /app

# Generate proto files directly in container
RUN mkdir -p /proto && \
    printf 'syntax = "proto3";\n\npackage logproto;\n\noption go_package = "stackmonitor.com/logproto";\n\nservice LogIngestion {\n  rpc StreamLogs(stream LogBatch) returns (stream Ack);\n}\n\nmessage LogBatch {\n  string agent_id = 1;\n  int64 batch_id = 2;\n  int64 timestamp_ms = 3;\n  repeated LogEntry logs = 4;\n  CompressionType compression = 5;\n  bytes compressed_payload = 6;\n  int32 original_size = 7;\n  map<string, string> metadata = 8;\n}\n\nmessage LogEntry {\n  int64 timestamp_ns = 1;\n  string level = 2;\n  string message = 3;\n  string source = 4;\n  map<string, string> fields = 5;\n  string agent_id = 6;\n}\n\nmessage Ack {\n  int64 batch_id = 1;\n  AckStatus status = 2;\n  string message = 3;\n  int64 server_timestamp_ms = 4;\n}\n\nenum AckStatus {\n  SUCCESS = 0;\n  RETRY = 1;\n  DROP = 2;\n}\n\nenum CompressionType {\n  NONE = 0;\n  ZSTD = 1;\n  LOGLITE = 2;\n}\n' > /proto/logs.proto && \
    printf 'syntax = "proto3";\n\npackage configproto;\n\noption go_package = "stackmonitor.com/configproto";\n\nservice ConfigService {\n  rpc GetConfig(ConfigRequest) returns (ConfigResponse);\n}\n\nmessage ConfigRequest {\n  string agent_id = 1;\n  string current_config_version = 2;\n}\n\nmessage ConfigResponse {\n  string config_version = 1;\n  bytes config_payload = 2;\n}\n' > /proto/config.proto

COPY go.mod go.sum ./

COPY . .

RUN apt-get update && apt-get install -y protobuf-compiler && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

RUN mkdir -p proto/logproto proto/configproto && \
    protoc --go_out=./proto/logproto --go_opt=paths=source_relative \
           --go-grpc_out=./proto/logproto --go-grpc_opt=paths=source_relative \
           --proto_path=/proto /proto/logs.proto && \
    protoc --go_out=./proto/configproto --go_opt=paths=source_relative \
           --go-grpc_out=./proto/configproto --go-grpc_opt=paths=source_relative \
           --proto_path=/proto /proto/config.proto && \
    echo 'module stackmonitor.com/logproto\n\ngo 1.21' > proto/logproto/go.mod && \
    echo 'module stackmonitor.com/configproto\n\ngo 1.21' > proto/configproto/go.mod

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/go-agent .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/go-agent .

CMD ["./go-agent"]
