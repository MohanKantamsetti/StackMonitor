module stackmonitor.com/ingestion-service

go 1.21

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.23.0
	github.com/klauspost/compress v1.17.8
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
)

replace stackmonitor.com/ingestion-service/proto/logproto => ./proto/logproto

