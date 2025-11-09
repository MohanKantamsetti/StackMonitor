-- ClickHouse Optimization Script for StackMonitor
-- Run this after initial setup to improve query performance

-- 1. Add Data Retention TTL (90 days)
-- Automatically removes old logs to prevent unbounded growth
ALTER TABLE stackmonitor.logs 
    MODIFY TTL timestamp + INTERVAL 90 DAY DELETE;

-- 2. Add Partitioning by Date
-- Improves query performance and enables efficient data pruning
ALTER TABLE stackmonitor.logs 
    MODIFY PARTITION BY toYYYYMMDD(timestamp);

-- 3. Create Materialized View for Error Statistics
-- Pre-aggregates error counts for fast dashboard queries
CREATE MATERIALIZED VIEW IF NOT EXISTS stackmonitor.error_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(hour)
ORDER BY (service, hour)
AS SELECT
    service,
    toStartOfHour(timestamp) as hour,
    count() as total_count,
    countIf(level = 'ERROR') as error_count,
    countIf(level = 'WARN') as warn_count,
    countIf(level = 'INFO') as info_count
FROM stackmonitor.logs
GROUP BY service, hour;

-- 4. Create Materialized View for Service Statistics
-- Optimizes per-service queries
CREATE MATERIALIZED VIEW IF NOT EXISTS stackmonitor.service_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(hour)
ORDER BY service
AS SELECT
    service,
    toStartOfHour(timestamp) as hour,
    count() as log_count,
    uniqExact(agent_id) as agent_count
FROM stackmonitor.logs
GROUP BY service, hour;

-- 5. Create Index on Message for Full-Text Search
-- Enables fast text search on log messages
-- Note: This creates a bloom filter index
ALTER TABLE stackmonitor.logs 
    ADD INDEX IF NOT EXISTS idx_message_bloom message TYPE bloom_filter GRANULARITY 1;

-- 6. Create Index on Service
-- Speeds up service-based filtering
ALTER TABLE stackmonitor.logs 
    ADD INDEX IF NOT EXISTS idx_service service TYPE set(100) GRANULARITY 1;

-- 7. Create Index on Level
-- Speeds up level-based filtering (ERROR, WARN, etc.)
ALTER TABLE stackmonitor.logs 
    ADD INDEX IF NOT EXISTS idx_level level TYPE set(10) GRANULARITY 1;

-- 8. Optimize Table to Apply Changes
-- Merges data parts and applies indexes
OPTIMIZE TABLE stackmonitor.logs FINAL;

-- 9. Create View for Recent Logs (Fast Query)
CREATE VIEW IF NOT EXISTS stackmonitor.recent_logs AS
SELECT *
FROM stackmonitor.logs
WHERE timestamp >= now() - INTERVAL 1 HOUR
ORDER BY timestamp DESC
LIMIT 1000;

-- 10. Create View for Error Summary
CREATE VIEW IF NOT EXISTS stackmonitor.error_summary AS
SELECT
    service,
    level,
    count() as count,
    any(message) as sample_message
FROM stackmonitor.logs
WHERE level IN ('ERROR', 'WARN')
  AND timestamp >= now() - INTERVAL 24 HOUR
GROUP BY service, level
ORDER BY count DESC;

-- Verification Queries

-- Check table size and compression
SELECT
    'Table Size' as metric,
    formatReadableSize(sum(bytes_on_disk)) as value
FROM system.parts
WHERE database = 'stackmonitor' AND table = 'logs' AND active;

-- Check index status
SELECT
    name,
    type,
    expr
FROM system.data_skipping_indices
WHERE database = 'stackmonitor' AND table = 'logs';

-- Check materialized views
SELECT
    name,
    engine
FROM system.tables
WHERE database = 'stackmonitor' AND engine LIKE '%MaterializedView%';

-- Test query performance (should be <50ms)
SELECT
    service,
    count() as log_count
FROM stackmonitor.logs
WHERE timestamp >= now() - INTERVAL 1 HOUR
  AND level = 'ERROR'
GROUP BY service
FORMAT PrettyCompact;

-- Show partition info
SELECT
    partition,
    name,
    rows,
    formatReadableSize(bytes_on_disk) as size
FROM system.parts
WHERE database = 'stackmonitor' AND table = 'logs' AND active
ORDER BY partition DESC
LIMIT 10;

