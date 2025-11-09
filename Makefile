.PHONY: help build up down restart logs clean health metrics test validate benchmark

# Default target
help:
	@echo "StackMonitor POC - Available Commands"
	@echo "======================================"
	@echo ""
	@echo "Setup & Deployment:"
	@echo "  make build        - Build all Docker images"
	@echo "  make up           - Start all services"
	@echo "  make down         - Stop all services"
	@echo "  make restart      - Restart all services"
	@echo "  make clean        - Stop and remove all containers, volumes, and data"
	@echo ""
	@echo "Monitoring & Health:"
	@echo "  make health       - Check health of all services"
	@echo "  make metrics      - Show metrics from all services"
	@echo "  make logs         - Follow logs from all services"
	@echo "  make logs-agent   - Follow Go agent logs"
	@echo "  make logs-python  - Follow Python agent logs"
	@echo "  make logs-ingest  - Follow ingestion service logs"
	@echo ""
	@echo "Testing & Validation:"
	@echo "  make test         - Run full validation suite"
	@echo "  make validate     - Alias for test"
	@echo "  make quick-test   - Run quick health check"
	@echo ""
	@echo "Database:"
	@echo "  make clickhouse   - Open ClickHouse client"
	@echo "  make db-stats     - Show database statistics"
	@echo "  make db-optimize  - Optimize ClickHouse tables"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Start in development mode (with logs)"
	@echo "  make rebuild      - Rebuild and restart specific service (e.g., make rebuild SERVICE=go-agent)"
	@echo ""

# Build all services
build:
	@echo "Building all Docker images..."
	docker-compose build

# Start all services
up:
	@echo "Starting all services..."
	docker-compose up -d
	@echo ""
	@echo "Services started! Waiting for initialization..."
	@sleep 5
	@echo ""
	@make health

# Stop all services
down:
	@echo "Stopping all services..."
	docker-compose down

# Restart all services
restart:
	@echo "Restarting all services..."
	docker-compose restart
	@echo "Services restarted!"

# View logs from all services
logs:
	docker-compose logs -f

# View logs from specific services
logs-agent:
	docker-compose logs -f go-agent

logs-python:
	docker-compose logs -f python-agent

logs-ingest:
	docker-compose logs -f ingestion-service

logs-api:
	docker-compose logs -f api-server

logs-gen:
	docker-compose logs -f log-generator

# Check health of all services
health:
	@echo "=== Service Health Status ==="
	@echo ""
	@echo "Go Agent (port 8081):"
	@curl -s http://localhost:8081/health 2>/dev/null | python -m json.tool || echo "  ❌ Not responding"
	@echo ""
	@echo "Python Agent (port 8083):"
	@curl -s http://localhost:8083/health 2>/dev/null | python -m json.tool || echo "  ❌ Not responding"
	@echo ""
	@echo "Ingestion Service (port 8082):"
	@curl -s http://localhost:8082/health 2>/dev/null | python -m json.tool || echo "  ❌ Not responding"
	@echo ""
	@echo "API Server (port 8080):"
	@curl -s http://localhost:8080/api/v1/logs/stats 2>/dev/null | python -m json.tool || echo "  ❌ Not responding"
	@echo ""

# Show metrics from all services
metrics:
	@echo "=== Go Agent Metrics ==="
	@curl -s http://localhost:8081/metrics 2>/dev/null | python -m json.tool || echo "❌ Not responding"
	@echo ""
	@echo "=== Python Agent Metrics ==="
	@curl -s http://localhost:8083/metrics 2>/dev/null | python -m json.tool || echo "❌ Not responding"
	@echo ""
	@echo "=== Ingestion Service Metrics ==="
	@curl -s http://localhost:8082/metrics 2>/dev/null | python -m json.tool || echo "❌ Not responding"
	@echo ""

# Run full validation suite
test:
	@echo "Running full validation suite..."
	python validate_stack_monitor.py

validate: test

# Quick health check
quick-test:
	@echo "Running quick health check..."
	@docker ps --filter "name=stackmonitor" --format "table {{.Names}}\t{{.Status}}"
	@echo ""
	@make health

# Open ClickHouse client
clickhouse:
	@echo "Opening ClickHouse client..."
	docker exec -it stackmonitor-poc-clickhouse-1 clickhouse-client

# Show database statistics
db-stats:
	@echo "=== ClickHouse Database Statistics ==="
	@docker exec stackmonitor-poc-clickhouse-1 clickhouse-client --query "\
		SELECT \
			'Total Logs' as metric, \
			formatReadableQuantity(count()) as value \
		FROM stackmonitor.logs \
		UNION ALL \
		SELECT \
			'Disk Usage' as metric, \
			formatReadableSize(sum(bytes_on_disk)) as value \
		FROM system.parts \
		WHERE database = 'stackmonitor' AND table = 'logs' AND active \
		UNION ALL \
		SELECT \
			'ERROR Logs' as metric, \
			formatReadableQuantity(count()) as value \
		FROM stackmonitor.logs WHERE level = 'ERROR' \
		UNION ALL \
		SELECT \
			'WARN Logs' as metric, \
			formatReadableQuantity(count()) as value \
		FROM stackmonitor.logs WHERE level = 'WARN' \
		UNION ALL \
		SELECT \
			'INFO Logs' as metric, \
			formatReadableQuantity(count()) as value \
		FROM stackmonitor.logs WHERE level = 'INFO' \
		FORMAT PrettyCompact"

# Optimize ClickHouse tables
db-optimize:
	@echo "Optimizing ClickHouse tables..."
	@docker exec stackmonitor-poc-clickhouse-1 clickhouse-client --query "OPTIMIZE TABLE stackmonitor.logs FINAL"
	@echo "Optimization complete!"

# Clean everything
clean:
	@echo "⚠️  This will remove all containers, volumes, and data!"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		echo "Cleaning up..."; \
		docker-compose down -v; \
		rm -rf clickhouse_data/ 2>/dev/null || true; \
		rm -f validation_results.json 2>/dev/null || true; \
		echo "Cleanup complete!"; \
	else \
		echo "Cancelled."; \
	fi

# Development mode (with logs)
dev:
	@echo "Starting in development mode..."
	docker-compose up

# Rebuild specific service
rebuild:
	@if [ -z "$(SERVICE)" ]; then \
		echo "❌ Error: SERVICE not specified"; \
		echo "Usage: make rebuild SERVICE=<service-name>"; \
		echo "Example: make rebuild SERVICE=go-agent"; \
		exit 1; \
	fi
	@echo "Rebuilding $(SERVICE)..."
	docker-compose build $(SERVICE)
	docker-compose up -d $(SERVICE)
	@echo "$(SERVICE) rebuilt and restarted!"

# Show all running containers
ps:
	@docker ps --filter "name=stackmonitor" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Show API documentation
docs:
	@echo "Starting HTTP server for API documentation..."
	@echo "Open your browser to: http://localhost:8888/api-docs.html"
	@python -m http.server 8888

# Quick start (build + up + test)
quickstart: build up
	@echo ""
	@echo "✅ StackMonitor is running!"
	@echo ""
	@echo "Access points:"
	@echo "  - UI:              http://localhost:3000"
	@echo "  - API:             http://localhost:8080/api/v1/logs"
	@echo "  - Swagger Docs:    Run 'make docs' and open http://localhost:8888/api-docs.html"
	@echo ""
	@echo "Run 'make test' to validate all scenarios"
	@echo ""
