#!/bin/bash
# Generate proto files for all services

set -e

echo "Building Protocol Buffers..."

# Create output directories
mkdir -p proto/go proto/python

# Generate Go proto files
echo "Generating Go code..."
protoc --go_out=proto/go --go_opt=paths=source_relative \
    --go-grpc_out=proto/go --go-grpc_opt=paths=source_relative \
    --proto_path=proto proto/logs.proto proto/config.proto

# Generate Python proto files  
echo "Generating Python code..."
python3 -m grpc_tools.protoc -Iproto --python_out=proto/python \
    --grpc_python_out=proto/python proto/logs.proto proto/config.proto

echo "Proto files generated successfully!"
echo "Go files in: proto/go/"
echo "Python files in: proto/python/"
