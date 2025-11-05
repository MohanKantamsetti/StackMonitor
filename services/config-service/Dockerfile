FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

RUN apt-get update && apt-get install -y protobuf-compiler && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Generate proto files inline - matching proto/config.proto exactly (including agent_id field)
RUN mkdir -p /proto && \
    printf '%s\n' 'syntax = "proto3";' 'package configproto;' 'option go_package = "stackmonitor.com/config-service/proto/configproto";' 'service ConfigService { rpc GetConfig(ConfigRequest) returns (ConfigResponse); }' 'message ConfigRequest { string agent_id = 1; string current_config_version = 2; }' 'message ConfigResponse { string config_version = 1; bytes config_payload = 2; }' > /proto/config.proto && \
    mkdir -p proto/configproto && \
    protoc --go_out=. --go_opt=module=stackmonitor.com/config-service \
           --go-grpc_out=. --go-grpc_opt=module=stackmonitor.com/config-service \
           --proto_path=/proto /proto/config.proto && \
    echo 'module stackmonitor.com/configproto' > proto/configproto/go.mod

COPY . .

# Update go.mod to use the generated proto
RUN echo 'replace stackmonitor.com/config-service/proto/configproto => ./proto/configproto' >> go.mod && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/config-service .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/config-service .

EXPOSE 8080

CMD ["./config-service"]
