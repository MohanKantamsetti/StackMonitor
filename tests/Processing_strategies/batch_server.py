#!/usr/bin/env python3
"""
Mock ingestion server that receives logs and measures metrics
"""

import socket
import time
import json
import argparse
import statistics
from collections import deque

class BatchServer:
    def __init__(self, strategy="fixed", port=5000):
        self.strategy = strategy
        self.port = port
        self.logs_received = 0
        self.latencies = deque(maxlen=10000)
        self.bandwidths = deque(maxlen=1000)
        self.start_time = time.time()
        self.batch_times = []
    
    def run(self):
        """Start server and listen for log batches"""
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        sock.bind(('localhost', self.port))
        sock.listen(1)
        
        print(f"Server listening on port {self.port} for {self.strategy} batches")
        
        try:
            conn, addr = sock.accept()
            print(f"Connection from {addr}")
            
            while time.time() - self.start_time < 70:  # Run for 70s
                try:
                    data = conn.recv(65536)  # Up to 64KB batch
                    if not data:
                        break
                    
                    # Parse batch metadata
                    batch_size = len(data)
                    batch_time = time.time()
                    logs_in_batch = batch_size // 100  # Assume ~100 bytes per log
                    
                    self.logs_received += logs_in_batch
                    self.bandwidths.append(batch_size)
                    self.batch_times.append(batch_time)
                    
                    # Simulate latency variation
                    latency = 10 + (batch_size / 100)  # ms
                    self.latencies.append(latency)
                    
                except Exception as e:
                    print(f"Error receiving data: {e}")
                    break
            
            conn.close()
        except Exception as e:
            print(f"Server error: {e}")
        finally:
            sock.close()
        
        self.print_metrics()
    
    def print_metrics(self):
        """Calculate and save metrics"""
        if not self.latencies or not self.bandwidths:
            print("No data received")
            return
        
        duration = time.time() - self.start_time
        total_bytes = sum(self.bandwidths)
        bandwidth_mbps = (total_bytes * 8) / (duration * 1_000_000)
        
        metrics = {
            "strategy": self.strategy,
            "logs_received": self.logs_received,
            "total_bytes": total_bytes,
            "bandwidth_mbps": round(bandwidth_mbps, 2),
            "latency_p50_ms": round(statistics.median(self.latencies), 2),
            "latency_p99_ms": round(statistics.quantiles(self.latencies, n=100)[98], 2),
            "latency_avg_ms": round(statistics.mean(self.latencies), 2),
            "batches_received": len(self.batch_times)
        }
        
        with open(f"metrics_{self.strategy}.json", "w") as f:
            json.dump(metrics, f, indent=2)
        
        print(f"\n✅ Metrics saved: {metrics}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--strategy", choices=["streaming", "fixed", "adaptive"], default="fixed")
    parser.add_argument("--output", default=None)
    args = parser.parse_args()
    
    server = BatchServer(strategy=args.strategy)
    server.run()
