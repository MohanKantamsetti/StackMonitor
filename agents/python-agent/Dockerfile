FROM python:3.10-slim AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

# Copy proto files (build context is project root)
COPY proto /proto

RUN mkdir -p /app/proto

RUN pip install grpcio-tools

RUN python -m grpc_tools.protoc -I/proto --python_out=/app/proto --grpc_python_out=/app/proto /proto/logs.proto /proto/config.proto

# Copy agent files
COPY agents/python-agent/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

COPY agents/python-agent/agent.py ./agent.py

FROM python:3.10-slim

WORKDIR /app

COPY --from=builder /app/proto /app/proto
COPY --from=builder /usr/local/lib/python3.10/site-packages /usr/local/lib/python3.10/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY --from=builder /app/agent.py ./agent.py

CMD ["python", "agent.py"]
