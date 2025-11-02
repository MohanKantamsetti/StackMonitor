FROM golang:1.21 AS builder

WORKDIR /app

# Generate proto file directly in container to avoid copy issues
RUN mkdir -p /proto && \
    echo 'syntax = "proto3";' > /proto/config.proto && \
    echo '' >> /proto/config.proto && \
    echo 'package configproto;' >> /proto/config.proto && \
    echo '' >> /proto/config.proto && \
    echo 'option go_package = "stackmonitor.com/configproto";' >> /proto/config.proto && \
    echo '' >> /proto/config.proto && \
    echo 'service ConfigService {' >> /proto/config.proto && \
    echo '  rpc GetConfig(ConfigRequest) returns (ConfigResponse);' >> /proto/config.proto && \
    echo '}' >> /proto/config.proto && \
    echo '' >> /proto/config.proto && \
    echo 'message ConfigRequest {' >> /proto/config.proto && \
    echo '  string agent_id = 1;' >> /proto/config.proto && \
    echo '  string current_config_version = 2;' >> /proto/config.proto && \
    echo '}' >> /proto/config.proto && \
    echo '' >> /proto/config.proto && \
    echo 'message ConfigResponse {' >> /proto/config.proto && \
    echo '  string config_version = 1;' >> /proto/config.proto && \
    echo '  bytes config_payload = 2;' >> /proto/config.proto && \
    echo '}' >> /proto/config.proto

COPY go.mod ./

COPY . .

RUN apt-get update && apt-get install -y protobuf-compiler && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

RUN mkdir -p proto && \
    protoc --go_out=./proto --go_opt=paths=source_relative \
           --go-grpc_out=./proto --go-grpc_opt=paths=source_relative \
           --proto_path=/proto /proto/config.proto && \
    echo 'module stackmonitor.com/configproto\n\ngo 1.21' > proto/go.mod

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/config-service .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/config-service .

EXPOSE 8080

CMD ["./config-service"]
