#!/usr/bin/env python3
"""
Design Space Exploration: Batching Strategies
Compares real-time streaming, fixed batching, and adaptive batching
Measures: CPU, memory, network bandwidth, latency
"""

import time
import psutil
import subprocess
import json
import statistics
from datetime import datetime

class BatchingTest:
    def __init__(self):
        self.process = None
        self.results = {}
    
    def measure_resources(self, pid, duration_sec=60):
        """Measure CPU, memory, network over duration"""
        cpu_samples = []
        memory_samples = []
        start_time = time.time()
        
        try:
            proc = psutil.Process(pid)
            while (time.time() - start_time) < duration_sec:
                try:
                    cpu_samples.append(proc.cpu_percent(interval=1))
                    memory_samples.append(proc.memory_info().rss / (1024*1024))  # MB
                except:
                    pass
                time.sleep(1)
        except:
            print(f"Failed to monitor process {pid}")
            return None
        
        return {
            "cpu_avg": statistics.mean(cpu_samples),
            "cpu_max": max(cpu_samples),
            "memory_avg": statistics.mean(memory_samples),
            "memory_max": max(memory_samples),
        }
    
    def run_test(self, strategy, duration=60, log_rate=5000):
        """
        Run a single batching strategy test
        Args:
            strategy: "streaming", "fixed", "adaptive"
            duration: test duration in seconds
            log_rate: logs per second to generate
        """
        print(f"\n{'='*60}")
        print(f"Testing: {strategy.upper()} Batching")
        print(f"{'='*60}")
        print(f"Duration: {duration}s, Log Rate: {log_rate} logs/sec")
        
        # Start server process (listening for logs)
        cmd = f"python3 batch_server.py --strategy {strategy} --output metrics_{strategy}.json"
        self.process = subprocess.Popen(cmd, shell=True)
        time.sleep(2)  # Wait for server startup
        
        # Get PID and start monitoring
        pid = self.process.pid
        print(f"Process PID: {pid}")
        
        # Start client (generator sending logs)
        client_cmd = f"python3 log_generator.py --rate {log_rate} --strategy {strategy} --duration {duration}"
        client = subprocess.Popen(client_cmd, shell=True)
        
        # Measure resources during test
        resources = self.measure_resources(pid, duration + 5)
        client.wait()
        self.process.terminate()
        
        # Read results from server
        try:
            with open(f"metrics_{strategy}.json") as f:
                metrics = json.load(f)
        except:
            metrics = {}
        
        result = {
            "strategy": strategy,
            "cpu_avg": resources["cpu_avg"] if resources else 0,
            "cpu_max": resources["cpu_max"] if resources else 0,
            "memory_avg": resources["memory_avg"] if resources else 0,
            "memory_max": resources["memory_max"] if resources else 0,
            **metrics
        }
        
        self.results[strategy] = result
        self._print_results(result)
        return result
    
    def _print_results(self, result):
        """Print formatted results"""
        print(f"\nResults for {result['strategy']}:")
        print(f"  CPU Usage:        {result['cpu_avg']:.2f}% avg, {result['cpu_max']:.2f}% max")
        print(f"  Memory:           {result['memory_avg']:.1f} MB avg, {result['memory_max']:.1f} MB max")
        print(f"  Network Bandwidth: {result.get('bandwidth_mbps', 'N/A')} Mbps")
        print(f"  Latency (p50):    {result.get('latency_p50_ms', 'N/A')} ms")
        print(f"  Latency (p99):    {result.get('latency_p99_ms', 'N/A')} ms")
        print(f"  Logs Dropped:     {result.get('logs_dropped', 0)}")
    
    def compare_results(self):
        """Compare all three strategies"""
        print(f"\n{'='*60}")
        print("COMPARISON TABLE")
        print(f"{'='*60}")
        
        print(f"\n{'Strategy':<15} {'CPU %':<10} {'Memory MB':<12} {'Bandwidth':<12} {'Latency p99':<15}")
        print("-" * 65)
        
        for strategy in ["streaming", "fixed", "adaptive"]:
            r = self.results.get(strategy, {})
            print(f"{strategy:<15} {r.get('cpu_avg', 0):<10.1f} {r.get('memory_avg', 0):<12.1f} "
                  f"{r.get('bandwidth_mbps', 'N/A'):<12} {r.get('latency_p99_ms', 'N/A'):<15}")

# Run tests
if __name__ == "__main__":
    tester = BatchingTest()
    
    for strategy in ["streaming", "fixed", "adaptive"]:
        tester.run_test(strategy, duration=60, log_rate=5000)
        time.sleep(5)  # Cool down between tests
    
    tester.compare_results()
    
    # Save summary
    with open("batching_comparison_results.json", "w") as f:
        json.dump(tester.results, f, indent=2)
    print("\n✅ Results saved to batching_comparison_results.json")
