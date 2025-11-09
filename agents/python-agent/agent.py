import grpc
import time
import yaml
import random
import re
import os
import queue
import threading
import signal
import zstandard as zstd
from datetime import datetime
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse
import json

# Import generated gRPC stubs (will be generated during build)
import sys
sys.path.insert(0, '/app/proto')
import logs_pb2
import logs_pb2_grpc
import config_pb2
import config_pb2_grpc

CONFIG_SERVICE = os.getenv("CONFIG_URL", "config-service:8080")
INGESTION_SERVICE = os.getenv("INGESTION_URL", "ingestion-service:50051")
LOG_FILES = ["/logs/application.log", "/logs/tomcat.log", "/logs/nginx.log"]
AGENT_ID = os.getenv("AGENT_ID", f"python-agent-{int(time.time())}")

# Regex patterns for different log formats
APP_LOG_REGEX = re.compile(r'^\[([^\]]+)\]\s+\[(\S+)\]\s+\[([^\]]+)\]\s+(.*)')  # Application log: [TIMESTAMP] [LEVEL] [SERVICE] MESSAGE
TOMCAT_LOG_REGEX = re.compile(r'^(\d{2}-[A-Za-z]{3}-\d{4}\s+\d{2}:\d{2}:\d{2}\.\d{3})\s+(\S+)\s+\[([^\]]+)\]\s+(.*)')  # Tomcat log
NGINX_LOG_REGEX = re.compile(r'^(\S+)\s+-\s+-\s+\[([^\]]+)\]\s+"(\S+)\s+(\S+)\s+(\S+)"\s+(\d+)\s+(\d+)\s+"([^"]+)"\s+"([^"]+)"')  # Nginx Combined

class AgentConfig:
    def __init__(self):
        self.version = ""
        self.base_rates = {"ERROR": 1.0, "WARN": 0.5, "INFO": 0.1, "DEBUG": 0.01}
        self.content_rules = []

    def load_from_yaml(self, yaml_content):
        data = yaml.safe_load(yaml_content)
        self.version = data.get("version", "")
        sampling = data.get("sampling", {})
        self.base_rates = sampling.get("base_rates", self.base_rates)
        self.content_rules = sampling.get("content_rules", [])

class MetricsHandler(BaseHTTPRequestHandler):
    agent = None  # Will be set by main
    
    def do_GET(self):
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/health':
            self.send_health()
        elif parsed_path.path == '/metrics':
            self.send_metrics()
        else:
            self.send_response(404)
            self.end_headers()
    
    def send_health(self):
        with self.agent.metrics_lock:
            uptime = time.time() - self.agent.start_time
            last_batch_ago = time.time() - self.agent.last_batch_time if self.agent.last_batch_time > 0 else uptime
            healthy = self.agent.healthy and last_batch_ago < 120  # 2 minutes
            
            response = {
                "status": "healthy" if healthy else "unhealthy",
                "agent_id": self.agent.agent_id,
                "uptime_seconds": uptime,
                "last_batch_ago": last_batch_ago,
                "config_version": self.agent.config_version,
                "log_queue_size": self.agent.log_queue.qsize()
            }
        
        status_code = 200 if healthy else 503
        self.send_response(status_code)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(response).encode())
    
    def send_metrics(self):
        with self.agent.metrics_lock:
            uptime = time.time() - self.agent.start_time
            compression_ratio = (self.agent.bytes_original / self.agent.bytes_compressed 
                               if self.agent.bytes_compressed > 0 else 1.0)
            
            response = {
                "agent_id": self.agent.agent_id,
                "uptime_seconds": uptime,
                "logs_processed": self.agent.logs_processed,
                "logs_sampled": self.agent.logs_sampled,
                "batches_sent": self.agent.batches_sent,
                "batches_failed": self.agent.batches_failed,
                "bytes_original": self.agent.bytes_original,
                "bytes_compressed": self.agent.bytes_compressed,
                "compression_ratio": compression_ratio,
                "logs_per_second": self.agent.logs_processed / uptime if uptime > 0 else 0,
                "log_queue_size": self.agent.log_queue.qsize()
            }
        
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(response).encode())
    
    def log_message(self, format, *args):
        # Suppress default logging
        pass

class LogHandler(FileSystemEventHandler):
    def __init__(self, agent, file_path):
        self.agent = agent
        self.file_path = file_path
        self.last_position = 0
        self._ensure_file()
        self._read_existing_logs()  # Read existing logs on startup

    def _ensure_file(self):
        if os.path.exists(self.file_path):
            self.last_position = 0  # Start from beginning to read existing logs

    def _read_existing_logs(self):
        """Read all existing logs from the file on startup (like Go agent)"""
        if not os.path.exists(self.file_path):
            return
        
        try:
            line_count = 0
            with open(self.file_path, 'r', encoding='utf-8', errors='ignore') as f:
                for line in f:
                    entry = self.agent.parse_log(line.strip(), self.file_path)
                    if entry:
                        self.agent.log_queue.put(entry)
                        line_count += 1
                self.last_position = f.tell()
            print(f"Processed {line_count} existing logs from {self.file_path}")
        except Exception as e:
            print(f"Error reading existing logs from {self.file_path}: {e}")

    def on_modified(self, event):
        if event.is_directory:
            return
        if event.src_path == self.file_path:
            self._read_new_lines()

    def _read_new_lines(self):
        try:
            with open(self.file_path, 'r', encoding='utf-8', errors='ignore') as f:
                f.seek(self.last_position)
                new_lines = f.readlines()
                self.last_position = f.tell()
                
                for line in new_lines:
                    entry = self.agent.parse_log(line.strip(), self.file_path)
                    if entry:
                        self.agent.log_queue.put(entry)
        except Exception as e:
            print(f"Error reading {self.file_path}: {e}")

class Agent:
    def __init__(self):
        self.agent_id = AGENT_ID
        self.config = AgentConfig()
        self.config_version = ""
        self.log_queue = queue.Queue()
        self.config_lock = threading.Lock()
        self.batch_id = 0
        
        # Metrics
        self.logs_processed = 0
        self.logs_sampled = 0
        self.batches_sent = 0
        self.batches_failed = 0
        self.bytes_original = 0
        self.bytes_compressed = 0
        self.start_time = time.time()
        self.last_batch_time = 0
        self.metrics_lock = threading.Lock()
        self.healthy = False
        
        # ZSTD compressor
        self.compressor = zstd.ZstdCompressor(level=3)

    def parse_log(self, line, source):
        if not line:
            return None

        t = None
        level = None
        service = None
        message = None

        # Try application log format first
        match = APP_LOG_REGEX.match(line)
        if match:
            timestamp_str, level, service, message = match.groups()
            try:
                if '.' in timestamp_str:
                    t = datetime.strptime(timestamp_str, "%Y-%m-%dT%H:%M:%S.%f")
                else:
                    t = datetime.strptime(timestamp_str, "%Y-%m-%dT%H:%M:%S")
            except:
                pass
        else:
            # Try Tomcat format
            match = TOMCAT_LOG_REGEX.match(line)
            if match:
                timestamp_str, level_str, thread, message = match.groups()
                try:
                    t = datetime.strptime(timestamp_str, "%d-%b-%Y %H:%M:%S.%f")
                    if level_str == "SEVERE":
                        level = "ERROR"
                    elif level_str == "WARNING":
                        level = "WARN"
                    else:
                        level = "INFO"
                    service = "tomcat"
                except:
                    pass
            else:
                # Try Nginx format
                match = NGINX_LOG_REGEX.match(line)
                if match:
                    client_ip, timestamp_str, method, path, protocol, status_code, body_bytes, referer, user_agent = match.groups()
                    try:
                        t = datetime.strptime(timestamp_str, "%d/%b/%Y:%H:%M:%S %z")
                        status_int = int(status_code)
                        if status_int >= 500:
                            level = "ERROR"
                        elif status_int >= 400:
                            level = "WARN"
                        else:
                            level = "INFO"
                        service = "nginx"
                        message = f"{method} {path} {protocol} - Status: {status_code}"
                    except:
                        pass

        if t is None or level is None:
            return None

        timestamp_ns = int(t.timestamp() * 1e9)

        # Apply sampling
        with self.config_lock:
            rate = self.config.base_rates.get(level, 0.1)
            # Check content rules
            for rule in self.config.content_rules:
                if rule.get("pattern", "") in message:
                    rate = rule.get("rate", rate)
                    break

        if rate < 1.0 and random.random() > rate:
            with self.metrics_lock:
                self.logs_sampled += 1
            return None  # Sampled out

        with self.metrics_lock:
            self.logs_processed += 1

        entry = logs_pb2.LogEntry(
            timestamp_ns=timestamp_ns,
            level=level,
            message=message,
            source=source,
            agent_id=self.agent_id
        )
        entry.fields["service"] = service
        entry.fields["trace_id"] = f"trace-{int(time.time() * 1e9)}"

        return entry

    def batch_sender(self, stub):
        buffer = []
        last_send = time.time()

        while True:
            try:
                # Collect entries with timeout
                entry = self.log_queue.get(timeout=1.0)
                buffer.append(entry)

                # Send if buffer full or timeout
                now = time.time()
                if len(buffer) >= 100 or (now - last_send) >= 10.0:
                    if buffer:
                        self._send_batch(stub, buffer)
                        buffer = []
                        last_send = now
            except queue.Empty:
                # Timeout - send if buffer has data
                if buffer and (time.time() - last_send) >= 10.0:
                    self._send_batch(stub, buffer)
                    buffer = []
                    last_send = time.time()

    def _send_batch(self, stub, logs):
        if not logs:
            return

        self.batch_id += 1
        
        # Serialize logs to bytes (same as Go agent)
        log_bytes = b''
        for log in logs:
            log_data = log.SerializeToString()
            log_bytes += log_data
        
        original_size = len(log_bytes)
        
        # Compress with ZSTD
        compressed = self.compressor.compress(log_bytes)
        compressed_size = len(compressed)
        
        # Create batch with compression (matching Go agent format)
        batch = logs_pb2.LogBatch(
            agent_id=self.agent_id,
            batch_id=self.batch_id,
            timestamp_ms=int(time.time() * 1000),
            logs=logs,  # Keep for backward compatibility
            compression=logs_pb2.CompressionType.ZSTD,
            compressed_payload=compressed,
            original_size=original_size,
        )

        try:
            stream = stub.StreamLogs(iter([batch]))
            # Get ack
            try:
                ack = next(stream)
                ratio = original_size / compressed_size if compressed_size > 0 else 1.0
                print(f"Sent batch {self.batch_id} with {len(logs)} logs (compressed {original_size}->{compressed_size} bytes, {ratio:.2f}x)")
                print(f"Received ack for batch {ack.batch_id}: {ack.message}")
                
                with self.metrics_lock:
                    self.batches_sent += 1
                    self.bytes_original += original_size
                    self.bytes_compressed += compressed_size
                    self.last_batch_time = time.time()
                    self.healthy = True
            except StopIteration:
                pass
        except Exception as e:
            print(f"Failed to send batch: {e}")
            with self.metrics_lock:
                self.batches_failed += 1

    def config_poller(self, stub):
        while True:
            try:
                with self.config_lock:
                    current_version = self.config_version

                request = config_pb2.ConfigRequest(
                    agent_id=self.agent_id,
                    current_config_version=current_version
                )

                response = stub.GetConfig(request)

                if response.config_version != current_version and response.config_payload:
                    new_config = AgentConfig()
                    new_config.load_from_yaml(response.config_payload.decode('utf-8'))

                    with self.config_lock:
                        self.config = new_config
                        self.config_version = response.config_version

                    print(f"Config reloaded to version {new_config.version}")

            except Exception as e:
                print(f"Failed to get config: {e}")

            time.sleep(60)  # Poll every 60 seconds

def main():
    agent = Agent()

    # Connect to services
    config_channel = grpc.insecure_channel(CONFIG_SERVICE)
    ingestion_channel = grpc.insecure_channel(INGESTION_SERVICE)

    config_stub = config_pb2_grpc.ConfigServiceStub(config_channel)
    ingestion_stub = logs_pb2_grpc.LogIngestionStub(ingestion_channel)

    # Initial config load
    try:
        request = config_pb2.ConfigRequest(agent_id=agent.agent_id, current_config_version="")
        response = config_stub.GetConfig(request)
        if response.config_payload:
            agent.config.load_from_yaml(response.config_payload.decode('utf-8'))
            agent.config_version = response.config_version
            print(f"Loaded initial config version: {response.config_version}")
    except Exception as e:
        print(f"Failed to load initial config: {e}")

    # Start config poller
    config_thread = threading.Thread(target=agent.config_poller, args=(config_stub,), daemon=True)
    config_thread.start()

    # Start batch sender
    batch_thread = threading.Thread(target=agent.batch_sender, args=(ingestion_stub,), daemon=True)
    batch_thread.start()

    # Start HTTP server for health and metrics
    MetricsHandler.agent = agent  # Set agent reference for HTTP handler
    http_port = int(os.getenv("HTTP_PORT", "8083"))
    http_server = HTTPServer(('0.0.0.0', http_port), MetricsHandler)
    http_thread = threading.Thread(target=http_server.serve_forever, daemon=True)
    http_thread.start()
    print(f"Starting HTTP server on port {http_port}")

    # Set up file watchers
    observer = Observer()
    for log_file in LOG_FILES:
        if os.path.exists(log_file):
            handler = LogHandler(agent, log_file)
            observer.schedule(handler, os.path.dirname(log_file), recursive=False)
            print(f"Started watching {log_file}")
        else:
            print(f"Log file {log_file} not found, skipping")

    observer.start()

    print("Python agent started. Waiting for logs...")
    
    # Setup graceful shutdown
    shutdown_event = threading.Event()
    
    def signal_handler(signum, frame):
        print("Shutdown signal received, gracefully stopping...")
        shutdown_event.set()
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    try:
        shutdown_event.wait()
    except KeyboardInterrupt:
        pass
    
    # Cleanup
    print("Stopping observer...")
    observer.stop()
    observer.join()
    
    print("Shutting down HTTP server...")
    http_server.shutdown()
    
    print("Closing gRPC channels...")
    config_channel.close()
    ingestion_channel.close()
    
    print("Python agent stopped gracefully")

if __name__ == "__main__":
    main()
