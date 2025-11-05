# StackMonitor PoC - Observability Platform

A Proof-of-Concept implementation of the StackMonitor observability platform as described in the M.Tech dissertation (S1-25_SEZG628T).

## Architecture Overview

The StackMonitor PoC consists of the following components:

- **Log Generators**: Synthetic log producers (Python)
- **Agents**: Lightweight collectors (Go and Python agents) that tail log files and forward to ingestion
- **Ingestion Service**: Central gRPC service that receives, deduplicates, and stores logs in ClickHouse
- **Config Service**: Serves configuration to agents via gRPC (supports hot-reload)
- **API Server**: REST/WebSocket API for querying logs and metrics
- **MCP Server**: Simulates LLM interface for natural language queries
- **AG-UI**: React-based frontend dashboard with AI chat interface

## Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- Node.js 18+ (for local development)

## Quick Start

1. **Clone and navigate to the repository:**
   ```bash
   cd stackmonitor-poc
   ```

2. **Start all services:**
   ```bash
   docker-compose up --build
   ```

3. **Access the UI:**
   - Open http://localhost:3000 in your browser
   - The AG-UI dashboard will show live logs, metrics, and an AI chat interface

## Services and Ports

| Service | Port | Description |
|---------|------|-------------|
| AG-UI | 3000 | React frontend dashboard |
| API Server | 5000 | REST API and WebSocket endpoints |
| MCP Server | 5001 | LLM query interface |
| Config Service | 8080 | Agent configuration gRPC service |
| Ingestion Service | 50051 | Log ingestion gRPC service |
| ClickHouse HTTP | 8123 | ClickHouse HTTP interface |
| ClickHouse Native | 9000 | ClickHouse native protocol |

## Project Structure

```
stackmonitor-poc/
├── agents/
│   ├── go-agent/          # Primary Go agent
│   └── python-agent/      # Compatibility Python agent
├── services/
│   ├── ingestion-service/ # Central log ingestion
│   ├── api-server/        # REST/WebSocket API
│   ├── config-service/    # Configuration service
│   └── mcp-server/        # MCP/LLM interface
├── log-generator/         # Synthetic log producer
├── ui/
│   └── ag-ui/            # React frontend
├── proto/                # Protocol Buffers definitions
├── config/               # Configuration files
└── docker-compose.yml    # Orchestration file
```

## Features Demonstrated

### 1. Agent Efficiency (Chapter 6.4, Scenario 1)
- Lightweight agents with minimal resource footprint
- Efficient gRPC streaming for log batches
- Adaptive sampling based on log levels

### 2. Hot-Reload Configuration (Scenario 2)
- Agents poll config service every 60 seconds
- Configuration changes are detected via version comparison
- Sampling rules update without agent restart

### 3. Deduplication (Scenario 3)
- In-memory deduplication cache in ingestion service
- Detects duplicate log entries based on timestamp + message + level
- Cache entries expire after 10 seconds

### 4. Natural Language Query (Scenario 4)
- MCP server provides intent recognition
- Calls appropriate REST API endpoints based on query
- Formats results for user-friendly display

## Configuration

### Log Generation

The log generator produces three types of logs:
- **Application logs** (`/logs/app.log`): Structured application logs with service, level, and message
- **Tomcat logs** (`/logs/tomcat.log`): Java application server logs with stack traces
- **Nginx logs** (`/logs/nginx.log`): HTTP access logs in Combined format

### Agent Configuration

Edit `config/config.yaml` to modify:
- Agent polling intervals
- Batch sizes and windows
- Sampling rates by log level
- Content-based sampling rules
- Retention policies

After editing, the config service will automatically detect changes (polling every 10 seconds) and agents will pick up new versions on their next poll.

### LLM Configuration

For natural language queries, the system uses **Google AI Studio (Gemini)**:

1. The API key is pre-configured in `docker-compose.yml` with your Google AI Studio key
2. LLM is enabled by default (`USE_LLM=true`)
3. The MCP server uses Gemini Pro for intent recognition and query understanding

**To customize:**
- Override the API key by setting `GEMINI_API_KEY` environment variable
- Disable LLM: Set `USE_LLM=false` in docker-compose.yml or `.env` file
- Without LLM, the system uses keyword matching (still functional but less intelligent)

## Development

### Building Proto Files

The proto files are automatically compiled during Docker builds. For local development:

```bash
# Install protoc and Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code (from service directory)
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       ../../proto/*.proto
```

### Running Services Individually

```bash
# Start only ClickHouse
docker-compose up clickhouse

# Start agents
docker-compose up go-agent python-agent

# View logs
docker-compose logs -f ingestion-service
```

## Testing Validation Scenarios

1. **Agent Efficiency**: Monitor resource usage with `docker stats`
2. **Hot-Reload**: Edit `config/config.yaml` and watch agent logs for config reload messages
3. **Deduplication**: Check ingestion service logs - duplicate messages should show "Processed X/Y logs" where Y > X
4. **Natural Language**: Open UI, use chat to ask "show me recent errors" or "what's the error rate?"

## Troubleshooting

### ClickHouse not accessible
- Wait for `clickhouse-init` to complete (check logs: `docker-compose logs clickhouse-init`)
- Verify ClickHouse health: `docker-compose ps clickhouse`

### Agents not sending logs
- Verify log files exist: `docker-compose exec go-agent ls -la /logs`
- Check agent logs: `docker-compose logs go-agent`
- Ensure ingestion service is running: `docker-compose ps ingestion-service`

### UI not connecting
- Verify API server is accessible: `curl http://localhost:5000/api/v1/logs`
- Check browser console for CORS/connection errors
- Ensure WebSocket proxy is configured in nginx (see Dockerfile)

## Notes

This is a Proof-of-Concept implementation:
- Deduplication uses simple in-memory cache (not production-ready)
- MCP server uses keyword matching (not true LLM)
- Compression is disabled for simplicity (ZSTD ready but commented out)
- Full Drain algorithm not implemented (basic pattern matching)
- ML-based sampling not implemented (static rules only)

## License

This PoC is for demonstration purposes as part of the M.Tech dissertation.

