#!/usr/bin/env python3
import os
import time
import re
import grpc
import json
from datetime import datetime
from collections import deque
import sys

# Add proto directory to path
sys.path.insert(0, '/app/proto')

try:
    import logs_pb2
    import logs_pb2_grpc
except ImportError as e:
    print(f"Failed to import proto modules: {e}")
    print("Available files in /app/proto:")
    if os.path.exists('/app/proto'):
        print(os.listdir('/app/proto'))
    time.sleep(60)
    sys.exit(1)

class PythonAgent:
    def __init__(self, agent_id, ingestion_url, log_paths):
        self.agent_id = agent_id
        self.ingestion_url = ingestion_url
        self.log_paths = log_paths
        self.buffer = deque(maxlen=100)
        self.batch_id = 0
        self.sample_rate = 0.1
        self.file_positions = {path: 0 for path in log_paths}
        
    def parse_log_line(self, line, source):
        """Parse a log line into structured format"""
        # Pattern: [timestamp] [level] [type] message
        pattern = r'\[(.*?)\]\s+\[(.*?)\]\s+\[(.*?)\]\s+(.*)'
        match = re.match(pattern, line.strip())
        
        if match:
            timestamp_str, level, log_type, message = match.groups()
            try:
                timestamp = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
                timestamp_ns = int(timestamp.timestamp() * 1e9)
            except:
                timestamp_ns = int(time.time() * 1e9)
            
            return {
                'timestamp_ns': timestamp_ns,
                'level': level,
                'message': message,
                'source': source,
                'fields': {'log_type': log_type}
            }
        else:
            # Fallback for unparseable lines
            return {
                'timestamp_ns': int(time.time() * 1e9),
                'level': 'INFO',
                'message': line.strip(),
                'source': source,
                'fields': {}
            }
    
    def tail_logs(self):
        """Tail log files and add entries to buffer"""
        for log_path in self.log_paths:
            if not os.path.exists(log_path):
                continue
            
            try:
                with open(log_path, 'r') as f:
                    # Seek to last known position
                    f.seek(self.file_positions[log_path])
                    
                    for line in f:
                        if not line.strip():
                            continue
                        
                        # Sample logs
                        import random
                        if random.random() > self.sample_rate:
                            continue
                        
                        source = os.path.basename(log_path)
                        log_entry = self.parse_log_line(line, source)
                        self.buffer.append(log_entry)
                    
                    # Update position
                    self.file_positions[log_path] = f.tell()
            
            except Exception as e:
                print(f"Error tailing {log_path}: {e}")
    
    def send_batch(self):
        """Send buffered logs to ingestion service"""
        if len(self.buffer) == 0:
            return
        
        try:
            channel = grpc.insecure_channel(self.ingestion_url)
            stub = logs_pb2_grpc.LogIngestionStub(channel)
            
            # Create batch
            self.batch_id += 1
            log_entries = []
            
            while self.buffer:
                entry_data = self.buffer.popleft()
                log_entry = logs_pb2.LogEntry(
                    timestamp_ns=entry_data['timestamp_ns'],
                    level=entry_data['level'],
                    message=entry_data['message'],
                    source=entry_data['source'],
                    agent_id=self.agent_id,
                    fields=entry_data['fields']
                )
                log_entries.append(log_entry)
            
            batch = logs_pb2.LogBatch(
                agent_id=self.agent_id,
                batch_id=self.batch_id,
                timestamp_ms=int(time.time() * 1000),
                logs=log_entries,
                compression=logs_pb2.CompressionType.NONE
            )
            
            # Send batch and get ack
            def request_iterator():
                yield batch
            
            responses = stub.StreamLogs(request_iterator())
            
            for response in responses:
                if response.status == logs_pb2.AckStatus.SUCCESS:
                    print(f"Batch {self.batch_id} sent successfully ({len(log_entries)} logs)")
                else:
                    print(f"Batch {self.batch_id} failed: {response.message}")
            
            channel.close()
            
        except Exception as e:
            print(f"Error sending batch: {e}")
    
    def run(self):
        """Main agent loop"""
        print(f"Python agent {self.agent_id} starting...")
        print(f"Monitoring logs: {', '.join(self.log_paths)}")
        print(f"Ingestion URL: {self.ingestion_url}")
        
        while True:
            try:
                # Tail logs
                self.tail_logs()
                
                # Send batch if buffer has enough entries
                if len(self.buffer) >= 10:
                    self.send_batch()
                
                time.sleep(1)
                
            except KeyboardInterrupt:
                print("\nShutting down agent...")
                if self.buffer:
                    self.send_batch()
                break
            except Exception as e:
                print(f"Error in agent loop: {e}")
                time.sleep(5)

def main():
    agent_id = os.environ.get('AGENT_ID', 'python-agent-1')
    ingestion_url = os.environ.get('INGESTION_URL', 'ingestion-service:50051')
    
    # Log paths to monitor
    log_paths = [
        '/logs/tomcat.log',
        '/logs/nginx.log',
        '/logs/application.log'
    ]
    
    agent = PythonAgent(agent_id, ingestion_url, log_paths)
    agent.run()

if __name__ == '__main__':
    main()

