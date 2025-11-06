#!/usr/bin/env python3
"""
Log generator that sends logs using different batching strategies
"""

import socket
import time
import argparse
import random

class LogGenerator:
    def __init__(self, strategy="fixed", rate=5000, duration=60):
        self.strategy = strategy
        self.rate = rate  # logs per second
        self.duration = duration
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    
    def connect(self):
        """Connect to batch server"""
        self.sock.connect(('localhost', 5000))
        print(f"Connected to server for {self.strategy} strategy")
    
    def generate_log(self, idx):
        """Generate a single log entry (~100 bytes)"""
        levels = ["INFO", "WARN", "ERROR"]
        level = random.choice(levels)
        return f"[{time.time():.3f}] {level} [service-{idx % 10}] Log message {idx}\n"
    
    def run_streaming(self):
        """Real-time streaming: send each log immediately"""
        print("Strategy: Real-time Streaming (send immediately)")
        start = time.time()
        interval = 1.0 / self.rate
        idx = 0
        
        while time.time() - start < self.duration:
            log = self.generate_log(idx)
            self.sock.sendall(log.encode())
            idx += 1
            time.sleep(interval)
        
        print(f"Streamed {idx} logs")
    
    def run_fixed_batching(self):
        """Fixed batching: send every 10 seconds or 64KB"""
        print("Strategy: Fixed Batching (10s window, 64KB max)")
        start = time.time()
        batch = []
        batch_size = 0
        last_send = time.time()
        idx = 0
        
        while time.time() - start < self.duration:
            log = self.generate_log(idx)
            batch.append(log)
            batch_size += len(log.encode())
            idx += 1
            
            # Send if: buffer full (64KB) OR time window (10s) elapsed
            if batch_size >= 65536 or (time.time() - last_send) >= 10:
                payload = "".join(batch).encode()
                self.sock.sendall(payload)
                batch = []
                batch_size = 0
                last_send = time.time()
            
            # Rate limit to target rate
            time.sleep(1.0 / self.rate)
        
        # Flush remaining
        if batch:
            self.sock.sendall("".join(batch).encode())
        
        print(f"Fixed batching sent {idx} logs in {(time.time()-start):.1f}s")
    
    def run_adaptive_batching(self):
        """Adaptive batching: adjust window based on log rate and resource"""
        print("Strategy: Adaptive Batching (5-30s window)")
        start = time.time()
        batch = []
        batch_size = 0
        base_window = 10  # seconds
        last_send = time.time()
        idx = 0
        
        while time.time() - start < self.duration:
            log = self.generate_log(idx)
            batch.append(log)
            batch_size += len(log.encode())
            idx += 1
            
            # Adaptive window: if high log rate, increase window; if many ERRORs, send faster
            current_rate = idx / (time.time() - start)
            rate_factor = min(current_rate / 1000, 2.0)  # Up to 2×
            adaptive_window = base_window * rate_factor
            
            error_count = sum(1 for l in batch if "ERROR" in l)
            if error_count > 0:
                adaptive_window = min(adaptive_window, 2)  # Fast path for errors
            
            # Send if: buffer full OR adaptive window expired
            if batch_size >= 65536 or (time.time() - last_send) >= adaptive_window:
                payload = "".join(batch).encode()
                self.sock.sendall(payload)
                batch = []
                batch_size = 0
                last_send = time.time()
            
            # Rate limit
            time.sleep(1.0 / self.rate)
        
        # Flush remaining
        if batch:
            self.sock.sendall("".join(batch).encode())
        
        print(f"Adaptive batching sent {idx} logs in {(time.time()-start):.1f}s")
    
    def run(self):
        """Execute the appropriate strategy"""
        self.connect()
        
        try:
            if self.strategy == "streaming":
                self.run_streaming()
            elif self.strategy == "fixed":
                self.run_fixed_batching()
            elif self.strategy == "adaptive":
                self.run_adaptive_batching()
        finally:
            self.sock.close()

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--strategy", choices=["streaming", "fixed", "adaptive"], default="fixed")
    parser.add_argument("--rate", type=int, default=5000)
    parser.add_argument("--duration", type=int, default=60)
    args = parser.parse_args()
    
    gen = LogGenerator(strategy=args.strategy, rate=args.rate, duration=args.duration)
    gen.run()
