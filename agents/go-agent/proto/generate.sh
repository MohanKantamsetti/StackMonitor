#!/bin/bash
# Generate Protocol Buffers for Go and Python

set -e

echo "Generating Protocol Buffers..."

# Install protoc if not available (check first)
if ! command -v protoc &> /dev/null; then
    echo "protoc not found. Please install Protocol Buffers compiler."
    echo "See: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# Generate Go code
echo "Generating Go code..."
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/*.proto

# Generate Python code (requires grpcio-tools)
if command -v python3 &> /dev/null; then
    echo "Generating Python code..."
    python3 -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. proto/*.proto
fi

echo "Proto generation complete!"

