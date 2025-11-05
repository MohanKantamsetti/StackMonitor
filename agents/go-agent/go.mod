module stackmonitor.com/go-agent

go 1.21

require (
	github.com/fsnotify/fsnotify v1.7.0
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v3 v3.0.1
)

replace stackmonitor.com/configproto => ./proto
replace stackmonitor.com/logproto => ./proto
