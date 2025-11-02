FROM python:3.10-slim AS builder
WORKDIR /app

# Generate proto files inline
RUN mkdir -p /proto && \
    printf 'syntax = "proto3";\n\npackage logproto;\n\nservice LogIngestion {\n  rpc StreamLogs(stream LogBatch) returns (stream Ack);\n}\n\nmessage LogBatch {\n  string agent_id = 1;\n  int64 batch_id = 2;\n  int64 timestamp_ms = 3;\n  repeated LogEntry logs = 4;\n  CompressionType compression = 5;\n  bytes compressed_payload = 6;\n  int32 original_size = 7;\n  map<string, string> metadata = 8;\n}\n\nmessage LogEntry {\n  int64 timestamp_ns = 1;\n  string level = 2;\n  string message = 3;\n  string source = 4;\n  map<string, string> fields = 5;\n  string agent_id = 6;\n}\n\nmessage Ack {\n  int64 batch_id = 1;\n  AckStatus status = 2;\n  string message = 3;\n  int64 server_timestamp_ms = 4;\n}\n\nenum AckStatus {\n  SUCCESS = 0;\n  RETRY = 1;\n  DROP = 2;\n}\n\nenum CompressionType {\n  NONE = 0;\n  ZSTD = 1;\n  LOGLITE = 2;\n}\n' > /proto/logs.proto && \
    printf 'syntax = "proto3";\n\npackage configproto;\n\nservice ConfigService {\n  rpc GetConfig(ConfigRequest) returns (ConfigResponse);\n}\n\nmessage ConfigRequest {\n  string agent_id = 1;\n  string current_config_version = 2;\n}\n\nmessage ConfigResponse {\n  string config_version = 1;\n  bytes config_payload = 2;\n}\n' > /proto/config.proto

RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /app/proto
RUN pip install grpcio-tools

RUN python -m grpc_tools.protoc -I/proto --python_out=/app/proto --grpc_python_out=/app/proto /proto/logs.proto /proto/config.proto

COPY requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

COPY agent.py ./agent.py
FROM python:3.10-slim
WORKDIR /app
COPY --from=builder /app/proto /app/proto
COPY --from=builder /usr/local/lib/python3.10/site-packages /usr/local/lib/python3.10/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY --from=builder /app/agent.py ./agent.py
CMD ["python", "agent.py"]