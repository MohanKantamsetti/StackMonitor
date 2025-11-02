module stackmonitor.com/config-service

go 1.21

require (
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
)

replace stackmonitor.com/configproto => ./proto