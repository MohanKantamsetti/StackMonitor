import grpc
import time
import yaml
import random
import re
import os
import queue
import threading
from datetime import datetime
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler

# Import generated gRPC stubs (will be generated during build)
import sys
sys.path.insert(0, '/app/proto')
import logs_pb2
import logs_pb2_grpc
import config_pb2
import config_pb2_grpc

CONFIG_SERVICE = os.getenv("CONFIG_URL", "config-service:8080")
INGESTION_SERVICE = os.getenv("INGESTION_URL", "ingestion-service:50051")
LOG_FILES = ["/logs/app.log", "/logs/tomcat.log", "/logs/nginx.log"]
AGENT_ID = os.getenv("AGENT_ID", f"python-agent-{int(time.time())}")

# Regex patterns for different log formats
APP_LOG_REGEX = re.compile(r'^(\S+)\s+(\S+)\s+\[(\S+)\]\s+(.*)')  # Application log
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

class LogHandler(FileSystemEventHandler):
    def __init__(self, agent, file_path):
        self.agent = agent
        self.file_path = file_path
        self.last_position = 0
        self._ensure_file()

    def _ensure_file(self):
        if os.path.exists(self.file_path):
            self.last_position = os.path.getsize(self.file_path)

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
                    t = datetime.strptime(timestamp_str, "%Y-%m-%dT%H:%M:%S.%fZ")
                else:
                    t = datetime.strptime(timestamp_str, "%Y-%m-%dT%H:%M:%SZ")
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
            return None  # Sampled out

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
        batch = logs_pb2.LogBatch(
            agent_id=self.agent_id,
            batch_id=self.batch_id,
            timestamp_ms=int(time.time() * 1000),
            logs=logs,
            compression=logs_pb2.CompressionType.NONE,  # Simplified for PoC
        )

        try:
            stream = stub.StreamLogs(iter([batch]))
            # Get ack
            try:
                ack = next(stream)
                print(f"Received ack for batch {ack.batch_id}: {ack.message}")
            except StopIteration:
                pass
        except Exception as e:
            print(f"Failed to send batch: {e}")

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
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        observer.stop()
    observer.join()

if __name__ == "__main__":
    main()
