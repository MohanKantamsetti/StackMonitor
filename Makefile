.PHONY: proto-build build test clean

# Generate proto files for all services
proto-build:
@echo "Generating Protocol Buffers..."
@mkdir -p proto/gen/go proto/gen/python
@protoc --go_out=proto/gen/go --go_opt=paths=source_relative \
--go-grpc_out=proto/gen/go --go-grpc_opt=paths=source_relative \
--python_out=proto/gen/python --grpc_python_out=proto/gen/python \
--proto_path=proto proto/*.proto
@echo "Proto files generated successfully!"

# Build all Docker images
build:
docker compose build

# Start all services
up:
docker compose up -d

# Stop all services
down:
docker compose down

# View logs
logs:
docker compose logs -f

# Clean up
clean:
docker compose down -v
docker system prune -f
