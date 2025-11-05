#!/bin/bash
set -e

echo "Waiting for ClickHouse to be ready..."
for i in {1..30}; do
    if clickhouse-client --host clickhouse --query "SELECT 1" 2>/dev/null; then
        echo "ClickHouse is ready!"
        break
    fi
    echo "Waiting for ClickHouse... ($i/30)"
    sleep 1
done

# Create database
echo "Creating database..."
clickhouse-client --host clickhouse --query "CREATE DATABASE IF NOT EXISTS stackmonitor"

# Create table
echo "Creating table..."
clickhouse-client --host clickhouse --query "
CREATE TABLE IF NOT EXISTS stackmonitor.logs (
    timestamp DateTime64(3),
    level String,
    service String,
    message String,
    trace_id String,
    agent_id String,
    metadata Map(String, String),
    INDEX message_idx message TYPE tokenbf_v1(10240, 3, 0) GRANULARITY 1
) ENGINE = MergeTree()
ORDER BY (timestamp, service)
TTL timestamp + INTERVAL 7 DAY
"

echo "ClickHouse database and table initialized successfully!"
echo "Verifying table exists..."
clickhouse-client --host clickhouse --query "SELECT count() FROM stackmonitor.logs"

