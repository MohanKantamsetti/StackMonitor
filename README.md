# StackMonitor PoC - Intelligent Log Monitoring & Analysis

A production-ready proof-of-concept system for intelligent log monitoring, analysis, and troubleshooting with AI-powered natural language querying.

![Status](https://img.shields.io/badge/status-production--ready-brightgreen)
![Docker](https://img.shields.io/badge/docker-compose-blue)
![Go](https://img.shields.io/badge/go-1.21-blue)
![Python](https://img.shields.io/badge/python-3.10-blue)
![React](https://img.shields.io/badge/react-18-blue)

## 🎯 Features

- **📊 Real-time Log Monitoring**: Multi-agent system collecting logs from multiple sources
- **🤖 AI-Powered Analysis**: Natural language queries with intelligent error categorization
- **🔍 Smart Troubleshooting**: Automatic error pattern recognition and recommendations
- **📈 Metrics & Trends**: Error rate tracking and visualization
- **💬 Interactive Chat UI**: Modern, responsive web interface
- **⚡ High Performance**: Built with Go, ClickHouse, and optimized for scale

## 🏗️ Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Log Generator│────>│   Agents    │────>│  Ingestion  │
│  (Python)   │     │(Go/Python)  │     │  Service    │
└─────────────┘     └─────────────┘     └─────────────┘
                                             │
                                             ▼
                                    ┌─────────────┐
                                    │  ClickHouse │
                                    │  Database   │
                                    └─────────────┘
                                             │
                    ┌────────────────────────┼────────────────────────┐
                    ▼                        ▼                        ▼
            ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
            │ API Server  │         │ MCP Server  │         │  Config     │
            │  (REST API) │         │  (AI/LLM)   │         │  Service    │
            └─────────────┘         └─────────────┘         └─────────────┘
                    │                        │
                    └────────────┬───────────┘
                                 ▼
                         ┌─────────────┐
                         │   Web UI    │
                         │   (React)   │
                         └─────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Git
- (Optional) Gemini API key for enhanced LLM features

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd stackmonitor-poc
   ```

2. **Configure environment (optional)**
   ```bash
   cp .env.example .env
   # Edit .env and add your GEMINI_API_KEY if desired
   ```

3. **Build and start all services**
   ```bash
   # Using Make (recommended)
   make build
   make up

   # Or using Docker Compose directly
   docker compose build
   docker compose up -d
   ```

4. **Access the application**
   - Web UI: http://localhost:3000
   - API Server: http://localhost:5000
   - MCP Server: http://localhost:5001
   - ClickHouse: http://localhost:8123

## 📖 Usage

### Web UI

1. Open http://localhost:3000 in your browser
2. Use the chat sidebar to ask questions:
   - "What are the recent errors?"
   - "How to fix errors?"
   - "Show me error rate trends"
   - "Describe my logs"

### API Endpoints

#### Get Logs
```bash
GET /api/v1/logs?limit=100&level=ERROR
```

#### Get Log Statistics
```bash
GET /api/v1/logs/stats
```

#### Get Error Rate Metrics
```bash
GET /api/v1/metrics/error-rate
```

#### MCP Query (Natural Language)
```bash
POST /mcp/query
Content-Type: application/json

{
  "query": "What are the recent errors?"
}
```

### Log Generator Configuration

Configure log generation rates via environment variables in `docker-compose.yml`:

```yaml
environment:
  - ERROR_RATE=0.20      # 20% error logs
  - WARN_RATE=0.25       # 25% warning logs
  - LOG_RATE=1.0         # 1 log per second
  - DEBUG_MODE=false     # Set to true for console output
```

## 🧩 Components

### Services

- **log-generator**: Generates realistic application logs (Tomcat, Nginx, Application)
- **go-agent**: Go-based log collector with file watching and sampling
- **python-agent**: Python-based log collector with watchdog
- **ingestion-service**: gRPC service receiving logs from agents and storing in ClickHouse
- **config-service**: Configuration management for agents (sample rates, buffer sizes)
- **api-server**: REST API for querying logs and metrics
- **mcp-server**: Model Context Protocol server with AI-powered analysis
- **ag-ui**: Modern React web interface

### Agents

Both Go and Python agents:
- Monitor log files in `/logs` directory
- Sample logs based on configured rates
- Batch and send logs via gRPC to ingestion service
- Support dynamic configuration updates

### Database

- **ClickHouse**: High-performance columnar database optimized for log storage and analytics

## 🤖 AI Features

### Intelligent Error Analysis

The system automatically:
- Categorizes errors by type (Connection, Memory, Database, etc.)
- Provides statistical analysis with percentages
- Suggests specific fixes for each error category
- Offers actionable troubleshooting steps

### Error Categories

- 🔌 Connection Issues
- 💾 Memory Problems
- 🗄️ Database Errors
- 📦 Request Size Limits
- 🌐 Server Errors
- 💳 Payment/Order Issues
- ☁️ Cloud Storage Issues
- 🐛 Application Bugs
- 🔐 SSL/Certificate Issues

## 📝 Make Commands

```bash
make build    # Build all Docker images
make up       # Start all services
make down     # Stop all services
make logs     # View logs from all services
make clean    # Stop services and clean up volumes
```

## 🔧 Configuration

### Environment Variables

See `.env.example` for all available configuration options.

### Log Generator

Adjust log generation in `docker-compose.yml`:
- `ERROR_RATE`: Percentage of error logs (0.0-1.0)
- `WARN_RATE`: Percentage of warning logs (0.0-1.0)
- `LOG_RATE`: Logs per second
- `DEBUG_MODE`: Enable console output

### Agent Configuration

Agents can be dynamically configured via the config service:
- Sample rate: Percentage of logs to process
- Buffer size: Number of logs to batch before sending

## 🧪 Testing

### Test Log Generation
```bash
docker logs stackmonitor-poc-log-generator-1 --tail 20
```

### Test Agent Activity
```bash
docker logs stackmonitor-poc-go-agent-1 --tail 20
```

### Query ClickHouse Directly
```bash
docker exec stackmonitor-poc-clickhouse-1 clickhouse-client --query "SELECT count() FROM logs"
```

### Test API Endpoints
```bash
curl http://localhost:5000/api/v1/logs/stats
```

## 📊 Monitoring

### View Service Status
```bash
docker compose ps
```

### View Logs
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f api-server
```

### Check ClickHouse Data
```bash
docker exec stackmonitor-poc-clickhouse-1 clickhouse-client --query "SELECT level, count() as count FROM logs GROUP BY level"
```

## 🛠️ Development

### Project Structure

```
stackmonitor-poc/
├── agents/
│   ├── go-agent/          # Go-based log collector
│   └── python-agent/      # Python-based log collector
├── services/
│   ├── api-server/        # REST API service
│   ├── ingestion-service/  # gRPC log ingestion
│   ├── config-service/    # Configuration management
│   └── mcp-server/        # AI-powered analysis
├── log-generator/          # Log generation service
├── ui/
│   └── ag-ui/             # React web interface
├── proto/                 # Protocol Buffer definitions
├── config/                # Configuration files
└── docker-compose.yml     # Service orchestration
```

### Adding New Features

1. **New Log Sources**: Add to `log-generator/generator.py`
2. **New Agents**: Follow pattern in `agents/go-agent` or `agents/python-agent`
3. **New API Endpoints**: Add routes in `services/api-server/main.go`
4. **UI Components**: Add React components in `ui/ag-ui/src/components`

## 🔒 Security Notes

- Default ClickHouse setup uses no password (development only)
- For production, set `CLICKHOUSE_PASSWORD` in `.env`
- Gemini API key is optional but recommended for best AI features
- API endpoints have CORS enabled (adjust for production)

## 🐛 Troubleshooting

### Services won't start
```bash
docker compose down -v
docker compose up --build
```

### No logs appearing
- Check log generator: `docker logs stackmonitor-poc-log-generator-1`
- Check agents: `docker logs stackmonitor-poc-go-agent-1`
- Verify ClickHouse: `docker exec stackmonitor-poc-clickhouse-1 clickhouse-client --query "SELECT count() FROM logs"`

### UI not loading
- Check nginx proxy: `docker logs stackmonitor-poc-ag-ui-1`
- Verify API server: `curl http://localhost:5000/api/v1/logs/stats`

### MCP server errors
- Check if Gemini API key is set (optional)
- Review logs: `docker logs stackmonitor-poc-mcp-server-1`

## 📚 Additional Documentation

- [MCP Integration Guide](./MCP_INTEGRATION.md) - Integration with Cursor Agent Mode
- [mcp-config.json](./mcp-config.json) - MCP server configuration

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- ClickHouse for high-performance log storage
- Google Gemini for LLM capabilities
- React for the modern UI framework

---

**Status**: Production Ready ✅  
**Last Updated**: 2025-11-02

