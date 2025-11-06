#!/usr/bin/env python3
"""
Real storage engine comparison using Docker containers
"""

import requests
import psycopg2
import json
import time
from datetime import datetime

class RealStorageTest:
    def __init__(self):
        self.results = {}
        self.test_logs = self.generate_test_logs()
    
    def generate_test_logs(self, count=10000):
        """Generate test logs"""
        logs = []
        for i in range(count):
            log = {
                "timestamp": f"2025-11-02T{i % 24:02d}:{(i // 60) % 60:02d}:{(i % 60):02d}Z",
                "level": ["INFO", "ERROR", "WARN", "DEBUG"][i % 4],
                "message": f"Test log {i}",
                "latency": i % 1000
            }
            logs.append(log)
        return logs
    
    def test_clickhouse_real(self):
        """Test real ClickHouse"""
        print("Testing ClickHouse...")
        try:
            # Create table
            requests.post(
                "http://localhost:8123",
                params={"query": """
                    CREATE TABLE IF NOT EXISTS logs (
                        timestamp String,
                        level String,
                        message String,
                        latency Int32
                    ) ENGINE = MergeTree() ORDER BY timestamp
                """}
            )
            
            # Insert logs
            start = time.time()
            for log in self.test_logs:
                requests.post(
                    "http://localhost:8123",
                    params={"query": f"""
                        INSERT INTO logs VALUES ('{log['timestamp']}', '{log['level']}', '{log['message']}', {log['latency']})
                    """}
                )
            elapsed = time.time() - start
            
            print(f"  ClickHouse write: {len(self.test_logs)/elapsed:.0f} logs/sec")
            return {"engine": "clickhouse_real", "throughput": int(len(self.test_logs)/elapsed)}
        except Exception as e:
            print(f"  ClickHouse error: {e}")
            return None
    
    def test_postgresql_real(self):
        """Test real PostgreSQL"""
        print("Testing PostgreSQL...")
        try:
            conn = psycopg2.connect(
                host="localhost", database="logs",
                user="postgres", password="postgres"
            )
            cur = conn.cursor()
            
            # Create table
            cur.execute("""
                CREATE TABLE IF NOT EXISTS logs (
                    timestamp TEXT, level TEXT, message TEXT, latency INT
                )
            """)
            
            # Insert logs
            start = time.time()
            for log in self.test_logs:
                cur.execute(
                    "INSERT INTO logs VALUES (%s, %s, %s, %s)",
                    (log['timestamp'], log['level'], log['message'], log['latency'])
                )
            conn.commit()
            elapsed = time.time() - start
            
            print(f"  PostgreSQL write: {len(self.test_logs)/elapsed:.0f} logs/sec")
            conn.close()
            return {"engine": "postgresql_real", "throughput": int(len(self.test_logs)/elapsed)}
        except Exception as e:
            print(f"  PostgreSQL error: {e}")
            return None
    
    def test_elasticsearch_real(self):
        """Test real Elasticsearch"""
        print("Testing Elasticsearch...")
        try:
            # Create index
            requests.put("http://localhost:9200/logs", json={
                "mappings": {
                    "properties": {
                        "timestamp": {"type": "keyword"},
                        "level": {"type": "keyword"},
                        "message": {"type": "text"},
                        "latency": {"type": "integer"}
                    }
                }
            })
            
            # Bulk insert
            start = time.time()
            bulk_data = ""
            for i, log in enumerate(self.test_logs):
                bulk_data += json.dumps({"index": {"_id": i}}) + "\n"
                bulk_data += json.dumps(log) + "\n"
            
            requests.post(
                "http://localhost:9200/logs/_bulk",
                data=bulk_data,
                headers={"Content-Type": "application/x-ndjson"}
            )
            elapsed = time.time() - start
            
            print(f"  Elasticsearch write: {len(self.test_logs)/elapsed:.0f} logs/sec")
            return {"engine": "elasticsearch_real", "throughput": int(len(self.test_logs)/elapsed)}
        except Exception as e:
            print(f"  Elasticsearch error: {e}")
            return None
    
    def run_all(self):
        """Run tests"""
        print("\n" + "="*90)
        print("REAL STORAGE ENGINE TEST (via Docker)")
        print("="*90)
        
        for test in [self.test_postgresql_real, self.test_clickhouse_real, self.test_elasticsearch_real]:
            result = test()
            if result:
                self.results[result['engine']] = result

if __name__ == "__main__":
    # Start Docker: docker-compose up -d
    suite = RealStorageTest()
    suite.run_all()
