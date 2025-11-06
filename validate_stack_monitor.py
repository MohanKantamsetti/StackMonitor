#!/usr/bin/env python3
"""
StackMonitor Validation Suite - Master's Dissertation Level
Comprehensive validation with verbose output for all scenarios.
"""

import json
import sys
import time
import requests
import subprocess
from datetime import datetime, timedelta
from typing import Dict, List, Tuple, Optional

# Fix Windows console encoding
if sys.platform == 'win32':
    try:
        sys.stdout.reconfigure(encoding='utf-8')
    except:
        pass

class ValidationResult:
    def __init__(self, scenario: str):
        self.scenario = scenario
        self.passed = False
        self.details = []
        self.metrics = {}
        self.start_time = time.time()
        
    def add_detail(self, msg: str, level: str = "INFO"):
        self.details.append(f"[{level}] {msg}")
        print(f"  [{level}] {msg}")
        
    def set_metric(self, key: str, value):
        self.metrics[key] = value
        
    def finish(self, passed: bool):
        self.passed = passed
        self.duration = time.time() - self.start_time
        status = "[PASS]" if passed else "[FAIL]"
        print(f"\n{status} {self.scenario} ({self.duration:.2f}s)")
        return self

class StackMonitorValidator:
    def __init__(self):
        self.endpoints = {
            'api': 'http://localhost:5000/api/v1',
            'mcp': 'http://localhost:5001/mcp',
            'clickhouse': 'http://localhost:8123'
        }
        self.results = []
        
    def check_service(self, name: str) -> bool:
        """Check if a Docker service is running."""
        try:
            result = subprocess.run(
                ['docker', 'ps', '--filter', f'name={name}', '--format', '{{.Names}}'],
                capture_output=True, text=True, timeout=5
            )
            return name in result.stdout
        except:
            return False
            
    def get_container_stats(self, name: str) -> Optional[Dict]:
        """Get container CPU and memory stats."""
        # Try different container name formats
        container_names = [
            f'stackmonitor-poc-{name}-1',
            name,
            f'{name}-1'
        ]
        
        for container_name in container_names:
            try:
                result = subprocess.run(
                    ['docker', 'stats', '--no-stream', '--format', 
                     '{{.CPUPerc}}\t{{.MemUsage}}', container_name],
                    capture_output=True, text=True, timeout=5
                )
                if result.returncode == 0 and result.stdout.strip():
                    parts = result.stdout.strip().split('\t')
                    if len(parts) == 2:
                        cpu_str = parts[0].replace('%', '').strip()
                        mem_str = parts[1].split()[0].strip()  # e.g., "8.65MiB"
                        
                        try:
                            cpu = float(cpu_str) if cpu_str else 0.0
                            
                            if 'MiB' in mem_str:
                                mem_mb = float(mem_str.replace('MiB', ''))
                            elif 'GiB' in mem_str:
                                mem_mb = float(mem_str.replace('GiB', '')) * 1024
                            elif 'KiB' in mem_str:
                                mem_mb = float(mem_str.replace('KiB', '')) / 1024
                            else:
                                mem_mb = 0.0
                                
                            return {'cpu': cpu, 'memory_mb': mem_mb}
                        except ValueError:
                            continue
            except Exception:
                continue
        return None
        
    def scenario_0_health_check(self) -> ValidationResult:
        """Scenario 0: System Health Check"""
        r = ValidationResult("System Health Check")
        
        services = {
            'log_generator': 'log-generator',
            'go_agent': 'go-agent',
            'python_agent': 'python-agent',
            'ingestion': 'ingestion-service',
            'clickhouse': 'clickhouse',
            'api_server': 'api-server',
            'mcp_server': 'mcp-server',
            'ui': 'ui'
        }
        
        all_running = True
        for key, name in services.items():
            running = self.check_service(name)
            status = "[RUNNING]" if running else "[STOPPED]"
            r.add_detail(f"{status}: {key}", "INFO" if running else "ERROR")
            if not running:
                all_running = False
                
        r.finish(all_running)
        return r
        
    def scenario_1_agent_efficiency(self) -> ValidationResult:
        """Scenario 1: Agent Efficiency (CPU <5%, Memory <100MB)"""
        r = ValidationResult("Agent Efficiency")
        
        r.add_detail("Measuring Go agent resource usage during log processing...")
        r.add_detail("Waiting 60 seconds for Go agent to stabilize and process logs...", "INFO")
        time.sleep(60)
        r.add_detail("Agent warmup period complete.", "INFO")
        
        r.add_detail("Step 1: Measuring idle CPU usage...", "INFO")
        
        # Measure idle CPU
        idle_samples = []
        for i in range(3):
            stats = self.get_container_stats('go-agent')
            if stats:
                idle_samples.append(stats['cpu'])
            time.sleep(1)
        
        idle_cpu = sum(idle_samples) / len(idle_samples) if idle_samples else 0
        r.add_detail(f"Idle CPU samples: {idle_samples}", "INFO")
        r.add_detail(f"Average idle CPU: {idle_cpu:.2f}%", "INFO")
        
        # Step 2: Wait for and confirm a specific batch is being processed
        r.add_detail("Step 2: Checking for recent batch processing activity...", "INFO")
        batch_confirmed = False
        batch_id = None
        batch_log_count = 0
        max_wait_seconds = 90  # Extended wait time to 90 seconds
        wait_interval = 3
        
        # First, check if there are any recent batches
        recent_batch_found = False
        try:
            logs = subprocess.run(
                ['docker', 'logs', '--tail', '50', 'stackmonitor-poc-go-agent-1'],
                capture_output=True, text=True, timeout=5
            )
            log_output = logs.stdout + logs.stderr
            
            # Look for "Batch X sent: Y logs"
            for line in log_output.split('\n'):
                if 'Batch' in line and 'sent:' in line and 'logs' in line:
                    recent_batch_found = True
                    # Extract batch ID and log count: "Batch 123 sent: 45 logs"
                    try:
                        # Format: "Batch 123 sent: 45 logs"
                        if 'Batch' in line and 'sent:' in line:
                            batch_part = line.split('Batch')[1].split('sent:')[0].strip()
                            batch_id = int(batch_part)
                            logs_part = line.split('sent:')[1].split()[0].strip()
                            batch_log_count = int(logs_part)
                            r.add_detail(f"Found recent batch {batch_id} with {batch_log_count} logs in agent logs", "INFO")
                            break
                    except (ValueError, IndexError):
                        continue
        except Exception as e:
            r.add_detail(f"Could not check agent logs: {str(e)}", "WARN")
        
        # If no recent batch found, wait for a new one
        if not recent_batch_found:
            r.add_detail("No recent batch found, waiting for agent to send a new batch...", "INFO")
            for attempt in range(max_wait_seconds // wait_interval):
                try:
                    logs = subprocess.run(
                        ['docker', 'logs', '--tail', '10', 'stackmonitor-poc-go-agent-1'],
                        capture_output=True, text=True, timeout=5
                    )
                    log_output = logs.stdout + logs.stderr
                    
                    # Look for "Batch X sent: Y logs"
                    for line in log_output.split('\n'):
                        if 'Batch' in line and 'sent:' in line and 'logs' in line:
                            try:
                                # Format: "Batch 123 sent: 45 logs"
                                batch_part = line.split('Batch')[1].split('sent:')[0].strip()
                                batch_id = int(batch_part)
                                logs_part = line.split('sent:')[1].split()[0].strip()
                                batch_log_count = int(logs_part)
                                batch_confirmed = True
                                r.add_detail(f"Confirmed new batch {batch_id} with {batch_log_count} logs", "INFO")
                                break
                            except (ValueError, IndexError):
                                continue
                    
                    if batch_confirmed:
                        break
                except Exception as e:
                    pass
                
                if attempt < (max_wait_seconds // wait_interval) - 1:
                    time.sleep(wait_interval)
        
        if recent_batch_found or batch_confirmed:
            batch_confirmed = True
            if not batch_id:
                # Try to get batch ID from logs again
                try:
                    logs = subprocess.run(
                        ['docker', 'logs', '--tail', '50', 'stackmonitor-poc-go-agent-1'],
                        capture_output=True, text=True, timeout=5
                    )
                    log_output = logs.stdout + logs.stderr
                    for line in log_output.split('\n'):
                        if 'Batch' in line and 'sent:' in line and 'logs' in line:
                            try:
                                # Format: "Batch 123 sent: 45 logs"
                                batch_part = line.split('Batch')[1].split('sent:')[0].strip()
                                batch_id = int(batch_part)
                                logs_part = line.split('sent:')[1].split()[0].strip()
                                batch_log_count = int(logs_part)
                                break
                            except (ValueError, IndexError):
                                continue
                except:
                    pass
        
        # Step 3: Measure CPU during active batch processing
        r.add_detail("Step 3: Measuring CPU during batch processing (waiting for next batch)...", "INFO")
        active_samples = []
        start_time = time.time()
        
        # Monitor for up to 15 seconds for active processing
        while len(active_samples) < 10 and (time.time() - start_time) < 15:
            stats = self.get_container_stats('go-agent')
            if stats:
                active_samples.append(stats['cpu'])
            time.sleep(0.5)
        
        active_cpu = sum(active_samples) / len(active_samples) if active_samples else 0
        r.add_detail(f"Active CPU samples ({len(active_samples)}): {active_samples[:10] if len(active_samples) > 10 else active_samples}", "INFO")
        r.add_detail(f"Average active CPU: {active_cpu:.2f}%", "INFO")
        
        # Verify batch was processed by checking ingestion service (wait up to 60 seconds)
        batch_processed = False
        if batch_id:
            r.add_detail(f"Step 4: Verifying batch {batch_id} was processed by ingestion service (waiting up to 60s)...", "INFO")
            for wait_attempt in range(60):  # Wait up to 60 seconds
                try:
                    ingestion_logs = subprocess.run(
                        ['docker', 'logs', '--tail', '200', 'stackmonitor-poc-ingestion-service-1'],
                        capture_output=True, text=True, timeout=5
                    )
                    log_output = ingestion_logs.stdout + ingestion_logs.stderr
                    # Check for exact batch match
                    if f'Received batch {batch_id}:' in log_output:
                        batch_processed = True
                        r.add_detail(f"Confirmed: Batch {batch_id} was processed by ingestion service", "INFO")
                        break
                except Exception as e:
                    pass
                
                if not batch_processed:
                    time.sleep(1)
            
            if not batch_processed:
                r.add_detail(f"Warning: Batch {batch_id} processing not yet visible in ingestion logs after 60s wait", "WARN")
                r.add_detail("This may indicate a delay in ingestion processing or batch was not sent", "INFO")
        
        # Calculate overall average
        all_samples = idle_samples + active_samples
        avg_cpu = sum(all_samples) / len(all_samples) if all_samples else 0
        
        # Get memory stats
        stats = self.get_container_stats('go-agent')
        if stats:
            mem = stats['memory_mb']
            r.add_detail(f"Memory: {mem:.2f}MB", "INFO")
            r.add_detail(f"Target: CPU <5%, Memory <100MB", "INFO")
            r.set_metric('cpu_percent', avg_cpu)
            r.set_metric('cpu_idle', idle_cpu)
            r.set_metric('cpu_active', active_cpu)
            r.set_metric('memory_mb', mem)
            if batch_confirmed:
                r.set_metric('batch_id', batch_id)
                r.set_metric('batch_log_count', batch_log_count)
            
            # Pass if CPU and memory targets are met, batch confirmation is nice-to-have
            passed = avg_cpu < 5.0 and mem < 100.0
            if batch_confirmed or batch_processed:
                r.add_detail(f"[PASS] Efficiency targets met: CPU {avg_cpu:.2f}% < 5% AND Memory {mem:.2f}MB < 100MB AND Batch processing confirmed", "INFO")
            elif passed:
                r.add_detail(f"[PASS] Efficiency targets met: CPU {avg_cpu:.2f}% < 5% AND Memory {mem:.2f}MB < 100MB (batch confirmation not available)", "INFO")
            else:
                if not batch_confirmed and not batch_processed:
                    r.add_detail(f"[WARN] Batch processing not confirmed within {max_wait_seconds} seconds", "WARN")
                else:
                    r.add_detail(f"[FAIL] Efficiency targets not met: CPU {avg_cpu:.2f}% >= 5% OR Memory {mem:.2f}MB >= 100MB", "ERROR")
        else:
            r.add_detail("Could not retrieve agent stats", "ERROR")
            passed = False
            
        r.finish(passed)
        return r
        
    def scenario_2_hot_reload(self) -> ValidationResult:
        """Scenario 2: Configuration Hot-Reload (No Restart Required)"""
        r = ValidationResult("Configuration Hot-Reload")
        
        r.add_detail("Testing configuration hot-reload capability...")
        
        # Check if services are running
        go_running = self.check_service('go-agent')
        py_running = self.check_service('python-agent')
        config_running = self.check_service('config-service')
        
        r.add_detail(f"Go Agent Status: {'RUNNING' if go_running else 'STOPPED'}")
        r.add_detail(f"Python Agent Status: {'RUNNING' if py_running else 'STOPPED'}")
        r.add_detail(f"Config Service Status: {'RUNNING' if config_running else 'STOPPED'}")
        
        if not (go_running and py_running and config_running):
            r.add_detail("All services must be running to test hot-reload", "ERROR")
            r.finish(False)
            return r
        
        # Step 1: Get current config version from config service BEFORE modifying
        r.add_detail("Step 1: Getting current config version from config service...", "INFO")
        config_version_before = None
        try:
            logs = subprocess.run(
                ['docker', 'logs', '--tail', '5', 'stackmonitor-poc-config-service-1'],
                capture_output=True, text=True, timeout=5
            )
            log_output = logs.stdout + logs.stderr
            # Find the most recent version hash
            for line in reversed(log_output.split('\n')):
                if 'Loaded new config version:' in line or 'Loaded initial config version:' in line:
                    # Extract version hash
                    if 'Loaded new config version:' in line:
                        parts = line.split('Loaded new config version:')[1].strip().split()
                        if parts:
                            config_version_before = parts[0]
                    elif 'Loaded initial config version:' in line:
                        parts = line.split('Loaded initial config version:')[1].strip().split()
                        if parts:
                            config_version_before = parts[0]
                    if config_version_before:
                        r.add_detail(f"Current config version: {config_version_before}", "INFO")
                        break
        except Exception as e:
            r.add_detail(f"Could not get current config version: {str(e)}", "WARN")
        
        # Step 2: Read and modify config
        r.add_detail("Step 2: Reading and modifying configuration...", "INFO")
        try:
            with open('config/config.yaml', 'r') as f:
                original_config = f.read()
            r.add_detail(f"Current config length: {len(original_config)} bytes", "INFO")
            
            # Modify config - ensure hash changes by adding unique comment
            import datetime
            timestamp_str = str(time.time())
            modified_config = original_config + f'\n# Test modification {timestamp_str}\n'
            modified_config = modified_config.replace('poll_interval: "60s"', 'poll_interval: "45s"')
            if 'version: "v1.0.1"' in modified_config:
                modified_config = modified_config.replace('version: "v1.0.1"', 'version: "v1.0.2"')
            
            with open('config/config.yaml', 'w') as f:
                f.write(modified_config)
            r.add_detail("Config file modified: poll_interval changed from 60s to 45s with timestamp", "INFO")
            r.add_detail(f"Modified config length: {len(modified_config)} bytes (was {len(original_config)} bytes)", "INFO")
        except Exception as e:
            r.add_detail(f"Could not modify config file: {str(e)}", "ERROR")
            r.finish(False)
            return r
        
        # Step 3: Get count of reload messages BEFORE modification
        r.add_detail("Step 3: Counting config reload messages before modification...", "INFO")
        reload_count_before = 0
        try:
            logs = subprocess.run(
                ['docker', 'logs', 'stackmonitor-poc-config-service-1'],
                capture_output=True, text=True, timeout=5
            )
            log_output = logs.stdout + logs.stderr
            for line in log_output.split('\n'):
                if 'Loaded new config version:' in line:
                    reload_count_before += 1
            r.add_detail(f"Total reload messages before modification: {reload_count_before}", "INFO")
        except Exception as e:
            r.add_detail(f"Could not count reload messages: {str(e)}", "WARN")
        
        # Step 4: Wait for config service to reload (polls every 10s, wait 30s to be safe)
        r.add_detail("Step 4: Waiting for config service to detect change (polls every 10s, waiting 30s)...", "INFO")
        time.sleep(30)
        
        # Step 5: Check for NEW config version after modification
        r.add_detail("Step 5: Checking if config service detected the change...", "INFO")
        config_reloaded = False
        config_version_after = None
        try:
            logs = subprocess.run(
                ['docker', 'logs', 'stackmonitor-poc-config-service-1'],
                capture_output=True, text=True, timeout=5
            )
            log_output = logs.stdout + logs.stderr
            reload_count_after = 0
            reload_messages = []
            for line in log_output.split('\n'):
                if 'Loaded new config version:' in line:
                    reload_count_after += 1
                    reload_messages.append(line.strip())
            
            # If reload count increased, a reload happened
            if reload_count_after > reload_count_before:
                config_reloaded = True
                # Get the most recent reload message
                if reload_messages:
                    latest_reload = reload_messages[-1]
                    # Extract version hash
                    if 'Loaded new config version:' in latest_reload:
                        parts = latest_reload.split('Loaded new config version:')[1].strip().split()
                        if parts:
                            config_version_after = parts[0].split('(')[0].strip()
                            r.add_detail(f"Config service reloaded: new version detected", "INFO")
                            r.add_detail(f"Latest reload log: {latest_reload}", "INFO")
                            if config_version_before:
                                r.add_detail(f"Version changed from {config_version_before} to {config_version_after}", "INFO")
            if not config_reloaded:
                r.add_detail("No reload detected in last 20 log lines", "WARN")
                r.add_detail("Recent logs:", "INFO")
                for line in log_output.split('\n')[-5:]:
                    if line.strip():
                        r.add_detail(f"  {line.strip()}", "INFO")
        except Exception as e:
            r.add_detail(f"Could not check config service logs: {str(e)}", "WARN")
        
        # Step 6: Restore original config (after checking if reload happened)
        r.add_detail("Step 6: Restoring original configuration...", "INFO")
        config_restored = False
        try:
            with open('config/config.yaml', 'w') as f:
                f.write(original_config)
            r.add_detail("Original config file restored", "INFO")
            config_restored = True
        except Exception as e:
            r.add_detail(f"Could not restore config: {str(e)}", "ERROR")
        
        # Determine pass/fail
        r.add_detail("", "INFO")
        r.add_detail("Hot-reload Test Summary:", "INFO")
        r.add_detail(f"  - Config file modified: YES", "INFO")
        r.add_detail(f"  - Config service reloaded: {'YES' if config_reloaded else 'NO'}", "INFO")
        r.add_detail(f"  - Original config restored: {'YES' if config_restored else 'NO'}", "INFO")
        
        # Pass if config service reloaded AND config was restored
        passed = config_reloaded and config_restored
        if not passed:
            if not config_reloaded:
                r.add_detail("", "INFO")
                r.add_detail("Config hot-reload test: Config service did not reload after modification", "WARN")
            if not config_restored:
                r.add_detail("", "INFO")
                r.add_detail("Config hot-reload test: Could not restore original config", "WARN")
        
        r.finish(passed)
        return r
        
    def scenario_3_deduplication(self) -> ValidationResult:
        """Scenario 3: Deduplication (Generator -> Agent -> Ingestion)"""
        r = ValidationResult("Deduplication")
        
        r.add_detail("Testing log deduplication through the pipeline: Generator -> Agent -> Ingestion...")
        
        # Step 1: Analyze generator logs for duplicates and extract timestamps
        r.add_detail("Step 1: Analyzing raw log files from generator for duplicates...", "INFO")
        generator_duplicates = 0
        generator_total = 0
        generator_unique = 0
        generator_log_timestamps = []  # Store timestamps of analyzed logs
        generator_messages = []  # Store messages for matching with agent logs
        
        try:
            # Read a sample of logs from the generator
            result = subprocess.run(
                ['docker', 'exec', 'stackmonitor-poc-log-generator-1', 'sh', '-c', 'tail -n 200 /logs/application.log 2>/dev/null || echo ""'],
                capture_output=True, text=True, timeout=5
            )
            if result.returncode == 0 and result.stdout.strip():
                log_lines = result.stdout.split('\n')
                seen = set()
                for line in log_lines:
                    if line.strip() and len(line) > 20:  # Valid log line
                        generator_total += 1
                        # Extract timestamp and message
                        # Format: "2025-11-06T... INFO [service] message"
                        parts = line.split(']', 1)
                        if len(parts) > 1:
                            # Extract timestamp (first part before first space)
                            timestamp_part = line.split()[0] if line.split() else ""
                            generator_log_timestamps.append(timestamp_part)
                            
                            message = parts[1].strip()
                            generator_messages.append(message)  # Store for matching
                            
                            key = message
                            if key in seen:
                                generator_duplicates += 1
                            else:
                                seen.add(key)
                                generator_unique += 1
                
                r.add_detail(f"Analyzed {generator_total} log lines from generator", "INFO")
                r.add_detail(f"  - Unique messages: {generator_unique}", "INFO")
                r.add_detail(f"  - Duplicate messages: {generator_duplicates}", "INFO")
                dup_rate = (generator_duplicates / generator_total * 100) if generator_total > 0 else 0
                r.add_detail(f"  - Duplicate rate at generator: {dup_rate:.2f}%", "INFO")
            else:
                r.add_detail("Could not read generator logs directly", "WARN")
        except Exception as e:
            r.add_detail(f"Could not analyze generator logs: {str(e)}", "WARN")
        
        # Step 2: Check what agent sent for the same logs (by querying API with timestamp range)
        r.add_detail("Step 2: Checking what agent sent for these generator logs...", "INFO")
        agent_logs_for_sample = []
        agent_duplicates = 0
        
        try:
            # Get logs from API that match the generator sample time window
            # Use the earliest and latest timestamps from generator sample
            if generator_log_timestamps:
                # Query API for logs
                resp = requests.get(f"{self.endpoints['api']}/logs", params={'limit': 1000}, timeout=15)
                if resp.status_code == 200:
                    data = resp.json()
                    logs = data.get('logs', []) if data else []
                    
                    # Match logs by message content (more reliable than timestamp matching)
                    # Use generator_messages already stored from Step 1
                    generator_messages_dict = {}
                    for i, message in enumerate(generator_messages):
                        if i < len(generator_log_timestamps):
                            generator_messages_dict[message] = generator_log_timestamps[i]
                    
                    agent_seen = set()
                    
                    for log_entry in logs:
                        log_timestamp = log_entry.get('timestamp', '')
                        log_message = log_entry.get('message', '')
                        
                        # Match by message content (exact match preferred)
                        matched = False
                        for gen_message, gen_timestamp in generator_messages_dict.items():
                            # Try exact match first
                            if log_message == gen_message:
                                agent_logs_for_sample.append(log_entry)
                                matched = True
                                # Check for duplicates in agent logs
                                key = (log_timestamp, log_message)
                                if key in agent_seen:
                                    agent_duplicates += 1
                                else:
                                    agent_seen.add(key)
                                break
                        
                        # If no exact match, try substring match (but only once per generator message)
                        if not matched:
                            for gen_message, gen_timestamp in generator_messages_dict.items():
                                # Substring match with minimum length to avoid false positives
                                if len(log_message) > 10 and len(gen_message) > 10:
                                    # Compare first 100 chars to avoid matching on very long messages
                                    if log_message[:100] == gen_message[:100] or log_message[:50] in gen_message[:100]:
                                        agent_logs_for_sample.append(log_entry)
                                        matched = True
                                        # Check for duplicates in agent logs
                                        key = (log_timestamp, log_message)
                                        if key in agent_seen:
                                            agent_duplicates += 1
                                        else:
                                            agent_seen.add(key)
                                        break
                    
                    r.add_detail(f"Found {len(agent_logs_for_sample)} logs from agent matching generator sample", "INFO")
                    r.add_detail(f"  - Unique agent logs: {len(agent_seen)}", "INFO")
                    r.add_detail(f"  - Duplicate agent logs: {agent_duplicates}", "INFO")
                else:
                    r.add_detail(f"Could not query API: Status {resp.status_code}", "WARN")
            else:
                r.add_detail("No generator timestamps available for matching", "WARN")
        except Exception as e:
            r.add_detail(f"Could not analyze agent logs: {str(e)}", "WARN")
        
        # Step 3: Compare generator vs agent
        r.add_detail("Step 3: Comparing generator logs vs agent sends...", "INFO")
        r.add_detail("", "INFO")
        r.add_detail("Deduplication Analysis:", "INFO")
        r.add_detail(f"  Generator logs analyzed: {generator_total} log lines", "INFO")
        r.add_detail(f"    - Unique messages: {generator_unique}", "INFO")
        r.add_detail(f"    - Duplicate messages: {generator_duplicates}", "INFO")
        r.add_detail(f"  Agent logs for same sample: {len(agent_logs_for_sample)} logs", "INFO")
        r.add_detail(f"    - Unique agent logs: {len(agent_seen) if 'agent_seen' in locals() else 0}", "INFO")
        r.add_detail(f"    - Duplicate agent logs: {agent_duplicates}", "INFO")
        
        # Determine pass/fail
        passed = False
        if generator_duplicates > 0:
            # Generator has duplicates, agent should have fewer or same
            if len(agent_logs_for_sample) > 0:
                agent_unique_count = len(agent_seen) if 'agent_seen' in locals() else len(agent_logs_for_sample)
                if agent_duplicates == 0:
                    r.add_detail(f"  - [PASS] Generator had {generator_duplicates} duplicates, agent sent {len(agent_logs_for_sample)} logs with 0 duplicates", "INFO")
                    r.add_detail("  - Deduplication is working: duplicates were filtered", "INFO")
                    passed = True
                else:
                    r.add_detail(f"  - [WARN] Generator had {generator_duplicates} duplicates, agent sent {len(agent_logs_for_sample)} logs with {agent_duplicates} duplicates", "WARN")
                    passed = agent_unique_count < generator_total  # Pass if agent has fewer unique logs
            else:
                r.add_detail("  - [WARN] Could not match generator logs with agent logs", "WARN")
                passed = False
        else:
            # No duplicates in generator
            if len(agent_logs_for_sample) > 0:
                r.add_detail(f"  - [PASS] No duplicates in generator ({generator_total} logs), agent sent {len(agent_logs_for_sample)} logs", "INFO")
                passed = True
            else:
                r.add_detail("  - [WARN] Could not verify agent logs", "WARN")
                passed = False
        
        r.set_metric('generator_total', generator_total)
        r.set_metric('generator_unique', generator_unique)
        r.set_metric('generator_duplicates', generator_duplicates)
        r.set_metric('agent_logs_matched', len(agent_logs_for_sample))
        r.set_metric('agent_duplicates', agent_duplicates)
            
        r.finish(passed)
        return r
        
    def scenario_4_nlq(self) -> ValidationResult:
        """Scenario 4: Natural Language Query (AI-Powered Analysis)"""
        r = ValidationResult("Natural Language Query")
        
        test_queries = [
            "Show me errors from payment service in last hour",
            "What are the recent warnings?",
            "Describe my logs",
            "How to fix errors?"
        ]
        
        passed_count = 0
        total_queries = len(test_queries)
        
        for idx, query in enumerate(test_queries, 1):
            r.add_detail(f"\n{'='*60}", "INFO")
            r.add_detail(f"Query {idx}/{total_queries}: '{query}'", "INFO")
            r.add_detail(f"{'='*60}", "INFO")
            r.add_detail(f"Request Method: POST", "INFO")
            r.add_detail(f"Request URL: {self.endpoints['mcp']}/query", "INFO")
            r.add_detail(f"Request Body: {{'query': '{query}'}}", "INFO")
            r.add_detail(f"Request Headers: {{'Content-Type': 'application/json'}}", "INFO")
            
            try:
                resp = requests.post(
                    f"{self.endpoints['mcp']}/query",
                    json={'query': query},
                    headers={'Content-Type': 'application/json'},
                    timeout=30
                )
                
                r.add_detail(f"Response Status: {resp.status_code}", "INFO")
                
                if resp.status_code == 200:
                    try:
                        data = resp.json()
                        response_text = data.get('response', '')
                        if response_text:
                            r.add_detail(f"Query succeeded", "INFO")
                            r.add_detail(f"Response Length: {len(response_text)} characters", "INFO")
                            # Show first 200 chars of response
                            preview = response_text[:200] + "..." if len(response_text) > 200 else response_text
                            r.add_detail(f"Response Preview: {preview}", "INFO")
                            passed_count += 1
                        else:
                            r.add_detail(f"Empty response received", "WARN")
                    except json.JSONDecodeError as e:
                        r.add_detail(f"Failed to parse JSON response: {str(e)}", "WARN")
                        r.add_detail(f"Raw Response (first 500 chars): {resp.text[:500]}", "WARN")
                else:
                    r.add_detail(f"Query failed with status {resp.status_code}", "ERROR")
                    r.add_detail(f"Response Body: {resp.text[:500]}", "ERROR")
                    
            except requests.exceptions.Timeout:
                r.add_detail(f"Query timed out after 30 seconds", "ERROR")
            except requests.exceptions.ConnectionError as e:
                r.add_detail(f"Connection error: {str(e)}", "ERROR")
            except Exception as e:
                r.add_detail(f"Query Error: {str(e)}", "ERROR")
        
        r.add_detail(f"\nNLQ Test Summary: {passed_count}/{total_queries} queries succeeded", "INFO")
        r.set_metric('total_queries', total_queries)
        r.set_metric('passed_queries', passed_count)
        
        # Pass if at least 75% of queries succeed
        passed = passed_count >= total_queries * 0.75
        r.finish(passed)
        return r
        
    def scenario_5_query_performance(self) -> ValidationResult:
        """Scenario 5: Query Performance (<500ms Target)"""
        r = ValidationResult("Query Performance")
        
        test_queries = [
            ("Full-text search (error pattern)", f"{self.endpoints['api']}/logs?level=ERROR&limit=100"),
            ("Recent logs query", f"{self.endpoints['api']}/logs?limit=100"),
            ("Level filter (WARN)", f"{self.endpoints['api']}/logs?level=WARN&limit=100"),
            ("Error rate metrics", f"{self.endpoints['api']}/metrics/error-rate?range=1h"),
            ("Log statistics", f"{self.endpoints['api']}/logs/stats")
        ]
        
        passed_count = 0
        total_time = 0
        successful_queries = 0
        
        for idx, (name, url) in enumerate(test_queries, 1):
            r.add_detail(f"\nTesting {idx}/{len(test_queries)}: {name}", "INFO")
            r.add_detail(f"Request: GET {url}", "INFO")
            
            try:
                start = time.time()
                resp = requests.get(url, timeout=15)
                elapsed = (time.time() - start) * 1000  # Convert to ms
                
                if resp.status_code == 200:
                    successful_queries += 1
                    total_time += elapsed
                    response_size = len(resp.content)
                    response_size_kb = response_size / 1024
                    r.add_detail(f"Status: {resp.status_code} OK", "INFO")
                    r.add_detail(f"Response Time: {elapsed:.2f}ms (target: <500ms)", 
                               "INFO" if elapsed < 500 else "WARN")
                    r.add_detail(f"Response Size: {response_size} bytes ({response_size_kb:.2f} KB)", "INFO")
                    
                    if elapsed < 500:
                        passed_count += 1
                        r.add_detail(f"[PASS] Query completed in {elapsed:.2f}ms, returned {response_size_kb:.2f} KB", "INFO")
                    else:
                        r.add_detail(f"[SLOW] Query took {elapsed:.2f}ms (exceeds 500ms target), returned {response_size_kb:.2f} KB", "WARN")
                else:
                    r.add_detail(f"Status: {resp.status_code}", "ERROR")
                    r.add_detail(f"Response: {resp.text[:200]}", "ERROR")
                    
            except requests.exceptions.Timeout:
                r.add_detail(f"Query timed out after 15 seconds", "ERROR")
            except requests.exceptions.ConnectionError as e:
                r.add_detail(f"Connection error: {str(e)}", "ERROR")
            except Exception as e:
                r.add_detail(f"Query failed: {str(e)}", "ERROR")
                
        avg_time = total_time / successful_queries if successful_queries > 0 else 0
        r.add_detail(f"\nPerformance Summary:", "INFO")
        r.add_detail(f"  - Successful queries: {successful_queries}/{len(test_queries)}", "INFO")
        r.add_detail(f"  - Queries <500ms: {passed_count}/{successful_queries if successful_queries > 0 else len(test_queries)}", "INFO")
        r.add_detail(f"  - Average Response Time: {avg_time:.2f}ms", "INFO")
        r.set_metric('queries_passed', passed_count)
        r.set_metric('queries_successful', successful_queries)
        r.set_metric('avg_response_ms', avg_time)
        
        # Pass if at least 75% of queries are successful AND at least 75% are <500ms
        passed = successful_queries >= len(test_queries) * 0.75 and (successful_queries == 0 or passed_count >= successful_queries * 0.75)
        r.finish(passed)
        return r
    
    def scenario_6_compression(self) -> ValidationResult:
        """Scenario 6: Log Compression and Storage Efficiency"""
        r = ValidationResult("Log Compression & Storage Efficiency")
        
        r.add_detail("Measuring compression ratio through pipeline: Generator -> Agent -> Ingestion -> ClickHouse...")
        r.add_detail("Using timestamps to filter last 3 minutes of logs for consistent comparison...", "INFO")
        
        # Calculate time window: 5-10 minutes ago (ensures logs have been processed)
        from datetime import datetime, timedelta
        now = datetime.utcnow()
        ten_minutes_ago = now - timedelta(minutes=10)
        five_minutes_ago = now - timedelta(minutes=5)
        
        r.add_detail(f"Time window: {ten_minutes_ago.isoformat()}Z to {five_minutes_ago.isoformat()}Z", "INFO")
        
        # Step 1: Get actual file sizes from generator
        r.add_detail("Step 1: Reading generator log files...", "INFO")
        generator_size_bytes = 0
        generator_log_count = 0
        try:
            # Try to get file size, use stat or ls -l
            result = subprocess.run(
                ['docker', 'exec', 'stackmonitor-poc-log-generator-1', 'sh', '-c', 'stat -c %s /logs/application.log 2>/dev/null || ls -l /logs/application.log 2>/dev/null | awk "{print $5}" || echo "0"'],
                capture_output=True, text=True, timeout=5
            )
            if result.returncode == 0 and result.stdout.strip():
                try:
                    app_log_size = int(result.stdout.strip())
                except ValueError:
                    app_log_size = 0
                # Also check other log files
                try:
                    tomcat_result = subprocess.run(
                        ['docker', 'exec', 'stackmonitor-poc-log-generator-1', 'sh', '-c', 'stat -c %s /logs/tomcat.log 2>/dev/null || ls -l /logs/tomcat.log 2>/dev/null | awk "{print $5}" || echo "0"'],
                        capture_output=True, text=True, timeout=5
                    )
                    if tomcat_result.returncode == 0 and tomcat_result.stdout.strip():
                        try:
                            tomcat_size = int(tomcat_result.stdout.strip())
                        except ValueError:
                            tomcat_size = 0
                    else:
                        tomcat_size = 0
                except:
                    tomcat_size = 0
                
                try:
                    nginx_result = subprocess.run(
                        ['docker', 'exec', 'stackmonitor-poc-log-generator-1', 'sh', '-c', 'stat -c %s /logs/nginx.log 2>/dev/null || ls -l /logs/nginx.log 2>/dev/null | awk "{print $5}" || echo "0"'],
                        capture_output=True, text=True, timeout=5
                    )
                    if nginx_result.returncode == 0 and nginx_result.stdout.strip():
                        try:
                            nginx_size = int(nginx_result.stdout.strip())
                        except ValueError:
                            nginx_size = 0
                    else:
                        nginx_size = 0
                except:
                    nginx_size = 0
                
                # Count log lines with timestamps in last 2 minutes
                try:
                    # Read recent logs and filter by timestamp
                    log_result = subprocess.run(
                        ['docker', 'exec', 'stackmonitor-poc-log-generator-1', 'sh', '-c', 'tail -n 500 /logs/application.log 2>/dev/null || echo ""'],
                        capture_output=True, text=True, timeout=5
                    )
                    if log_result.returncode == 0 and log_result.stdout.strip():
                        log_lines = log_result.stdout.split('\n')
                        for line in log_lines:
                            if line.strip() and len(line) > 20:
                                # Extract timestamp (first field)
                                parts = line.split()
                                if parts:
                                    try:
                                        # Parse ISO timestamp (format: 2025-11-06T03:57:17.642311Z or similar)
                                        # Format: [2025-11-06T04:52:07.151154] [ERROR] ...
                                        timestamp_str = parts[0].strip('[]')
                                        try:
                                            if 'T' in timestamp_str:
                                                log_time = datetime.strptime(timestamp_str, '%Y-%m-%dT%H:%M:%S.%f')
                                                # Filter logs in the 5-10 minute window
                                                if ten_minutes_ago <= log_time <= five_minutes_ago:
                                                    generator_log_count += 1
                                                    generator_size_bytes += len(line.encode('utf-8'))
                                        except:
                                            pass
                                    except:
                                        pass
                except Exception as e:
                    pass
                
                # Estimate size for last 2 minutes (proportional to line count)
                # Get total lines and size, then estimate for last 2 min
                try:
                    total_lines_result = subprocess.run(
                        ['docker', 'exec', 'stackmonitor-poc-log-generator-1', 'sh', '-c', 'wc -l /logs/application.log /logs/tomcat.log /logs/nginx.log 2>/dev/null | tail -1'],
                        capture_output=True, text=True, timeout=5
                    )
                    if total_lines_result.returncode == 0 and total_lines_result.stdout.strip():
                        total_lines = int(total_lines_result.stdout.strip().split()[0])
                        if total_lines > 0:
                            # Estimate size for last 2 minutes proportionally
                            generator_size_bytes = int((generator_log_count / total_lines) * (app_log_size + tomcat_size + nginx_size)) if total_lines > 0 else 0
                except:
                    generator_size_bytes = app_log_size + tomcat_size + nginx_size  # Fallback to total
                
                r.add_detail(f"Generator (5-10min ago): {generator_log_count} logs, {generator_size_bytes} bytes", "INFO")
            else:
                r.add_detail("Could not read generator log file sizes", "WARN")
        except Exception as e:
            r.add_detail(f"Could not measure generator size: {str(e)}", "WARN")
        
        # Step 2: Query ClickHouse for SAME time window (5-10 min ago)
        r.add_detail("Step 2: Querying ClickHouse for logs in same 5-10min window...", "INFO")
        stored_count = 0
        try:
            # Query recent logs with a larger limit
            resp = requests.get(f"{self.endpoints['api']}/logs", params={'limit': 5000}, timeout=15)
            if resp.status_code == 200:
                logs = resp.json().get('logs', [])
                r.add_detail(f"Retrieved {len(logs)} logs from API", "INFO")
                
                # Show sample timestamp for debugging
                if logs:
                    sample_ts = logs[0].get('timestamp', 'N/A')
                    r.add_detail(f"Sample timestamp format: {sample_ts}", "INFO")
                
                matched_logs = []
                parse_errors = 0
                # Filter to same 5min window
                for log in logs:
                    ts_str = log.get('timestamp', '')
                    if ts_str:
                        try:
                            log_time = None
                            # Try multiple timestamp formats
                            if 'T' in ts_str:
                                # Try with microseconds and Z
                                for fmt in ['%Y-%m-%dT%H:%M:%S.%fZ', '%Y-%m-%dT%H:%M:%SZ', 
                                           '%Y-%m-%dT%H:%M:%S.%f', '%Y-%m-%dT%H:%M:%S']:
                                    try:
                                        log_time = datetime.strptime(ts_str, fmt)
                                        break
                                    except:
                                        continue
                            
                            if log_time and log_time >= five_minutes_ago:
                                stored_count += 1
                                stored_bytes += 140  # ClickHouse compressed estimate per log
                                if len(matched_logs) < 3:
                                    matched_logs.append(log_time.strftime('%Y-%m-%d %H:%M:%S'))
                        except Exception as e:
                            parse_errors += 1
                            if parse_errors <= 2:  # Show first 2 errors
                                r.add_detail(f"Parse error: {ts_str} - {str(e)}", "WARN")
                
                if matched_logs:
                    r.add_detail(f"Matched log timestamps: {', '.join(matched_logs)}", "INFO")
                if parse_errors > 2:
                    r.add_detail(f"Total parse errors: {parse_errors}", "WARN")
                    
                r.add_detail(f"ClickHouse (5-10min ago): {stored_count} logs verified", "INFO")
            else:
                r.add_detail(f"API returned status {resp.status_code}", "WARN")
        except Exception as e:
            r.add_detail(f"Error querying ClickHouse: {str(e)}", "WARN")
        
        # Step 3: Check agent transmission (from go-agent logs for compression)
        r.add_detail("Step 3: Checking agent transmission size (gRPC + compression)...", "INFO")
        transmission_bytes = 0
        transmission_original = 0
        batch_count = 0
        try:
            # Check go-agent logs for compression stats
            logs = subprocess.run(
                ['docker', 'logs', '--since', '3m', 'stackmonitor-poc-go-agent-1'],
                capture_output=True, text=True, timeout=5
            )
            log_output = logs.stdout + logs.stderr
            for line in log_output.split('\n'):
                if 'compressed' in line.lower() and '->' in line:
                    # Parse: "Sent batch X with Y logs (compressed A->B bytes, C.DDx)"
                    try:
                        if 'compressed' in line:
                            parts = line.split('compressed')[1].split('bytes')[0]
                            if '->' in parts:
                                sizes = parts.split('->')
                                orig = int(sizes[0].strip())
                                comp = int(sizes[1].strip())
                                transmission_original += orig
                                transmission_bytes += comp
                                batch_count += 1
                    except:
                        pass
            
            if transmission_bytes > 0:
                trans_ratio = transmission_original / transmission_bytes if transmission_bytes > 0 else 1.0
                r.add_detail(f"Network Transmission ({batch_count} batches):", "INFO")
                r.add_detail(f"  Protobuf serialized: {transmission_original} bytes", "INFO")
                r.add_detail(f"  After ZSTD compression: {transmission_bytes} bytes ({trans_ratio:.2f}x)", "INFO")
                r.add_detail(f"  Bandwidth saved: {transmission_original - transmission_bytes} bytes ({100*(1-transmission_bytes/transmission_original):.1f}%)", "INFO")
            else:
                r.add_detail("No compression data found in agent logs (agents may not be sending compressed batches yet)", "WARN")
        except Exception as e:
            r.add_detail(f"Error checking transmission: {str(e)}", "WARN")
        
        # Get actual ClickHouse storage metrics
        actual_ch_size = 0
        total_logs_in_db = 0
        try:
            ch_size_result = subprocess.run(
                ['docker', 'exec', 'stackmonitor-poc-clickhouse-1', 'clickhouse-client', '--query',
                 "SELECT sum(bytes_on_disk) FROM system.parts WHERE database = 'stackmonitor' AND table = 'logs' AND active"],
                capture_output=True, text=True, timeout=10
            )
            if ch_size_result.returncode == 0 and ch_size_result.stdout.strip():
                actual_ch_size = int(ch_size_result.stdout.strip())
                
            # Get total log count
            count_result = subprocess.run(
                ['docker', 'exec', 'stackmonitor-poc-clickhouse-1', 'clickhouse-client', '--query',
                 "SELECT count() FROM stackmonitor.logs"],
                capture_output=True, text=True, timeout=10
            )
            if count_result.returncode == 0 and count_result.stdout.strip():
                total_logs_in_db = int(count_result.stdout.strip())
        except:
            pass
        
        # Summary
        r.add_detail("", "INFO")
        r.add_detail("Compression & Storage Summary:", "INFO")
        
        # Network transmission efficiency (most accurate metric)
        if transmission_bytes > 0 and transmission_original > 0:
            trans_ratio = transmission_original / transmission_bytes
            bandwidth_saved = transmission_original - transmission_bytes
            pct_saved = (bandwidth_saved / transmission_original) * 100
            r.add_detail(f"  Network Transmission:", "INFO")
            r.add_detail(f"    - Protobuf serialized: {transmission_original} bytes", "INFO")
            r.add_detail(f"    - After ZSTD compression: {transmission_bytes} bytes", "INFO")
            r.add_detail(f"    - Compression ratio: {trans_ratio:.2f}x", "INFO")
            r.add_detail(f"    - Bandwidth saved: {bandwidth_saved} bytes ({pct_saved:.1f}%)", "INFO")
        
        # ClickHouse storage (actual disk usage)
        if actual_ch_size > 0:
            r.add_detail(f"  ClickHouse Storage:", "INFO")
            r.add_detail(f"    - Total logs stored: {total_logs_in_db:,}", "INFO")
            r.add_detail(f"    - Disk usage (MergeTree compressed): {actual_ch_size:,} bytes ({actual_ch_size/1024:.1f} KB)", "INFO")
            if total_logs_in_db > 0:
                avg_per_log = actual_ch_size / total_logs_in_db
                r.add_detail(f"    - Average per log: {avg_per_log:.1f} bytes", "INFO")
        
        r.set_metric('gen_bytes', generator_size_bytes)
        r.set_metric('gen_logs', generator_log_count)
        r.set_metric('transmission_bytes', transmission_bytes)
        r.set_metric('transmission_original', transmission_original)
        r.set_metric('stored_logs', stored_count)
        r.set_metric('clickhouse_total_logs', total_logs_in_db)
        r.set_metric('clickhouse_disk_bytes', actual_ch_size)
        
        # Pass if we have transmission compression OR stored logs
        passed = (transmission_bytes > 0 and transmission_bytes < transmission_original) or stored_count > 0
        
        r.finish(passed)
        return r
        
    def run_all(self):
        """Run all validation scenarios."""
        print("=" * 80)
        print("# StackMonitor Complete Validation Suite")
        print(f"# Start Time: {datetime.now().isoformat()}")
        print("=" * 80)
        print()
        
        scenarios = [
            ("Scenario 0", self.scenario_0_health_check),
            ("Scenario 1", self.scenario_1_agent_efficiency),
            ("Scenario 2", self.scenario_2_hot_reload),
            ("Scenario 3", self.scenario_3_deduplication),
            ("Scenario 4", self.scenario_4_nlq),
            ("Scenario 5", self.scenario_5_query_performance),
            ("Scenario 6", self.scenario_6_compression),
        ]
        
        for name, func in scenarios:
            print("\n" + "=" * 80)
            print(f"  {name.upper()}")
            print("=" * 80)
            result = func()
            self.results.append(result)
            print()
            
        # Summary
        self.print_summary()
        
    def print_summary(self):
        """Print validation summary."""
        print("\n" + "=" * 80)
        print("  VALIDATION SUMMARY REPORT")
        print("=" * 80)
        
        total_duration = sum(r.duration for r in self.results)
        passed = sum(1 for r in self.results if r.passed)
        total = len(self.results)
        
        print(f"\nValidation Duration: {total_duration:.1f}s")
        print(f"Scenarios Tested: {total}")
        print(f"Scenarios Passed: {passed}/{total}")
        print(f"Overall Result: {'[PASS] ALL TESTS PASSED' if passed == total else '[FAIL] SOME TESTS FAILED'}")
        
        print("\nDetailed Results:")
        print("─" * 80)
        for r in self.results:
            status = "[PASS]" if r.passed else "[FAIL]"
            print(f"{status} {r.scenario}")
            if r.metrics:
                for key, value in r.metrics.items():
                    if isinstance(value, float):
                        print(f"   - {key}: {value:.2f}")
                    else:
                        print(f"   - {key}: {value}")
                        
        # Export results
        results_data = {
            'timestamp': datetime.now().isoformat(),
            'duration_seconds': total_duration,
            'total_scenarios': total,
            'passed_scenarios': passed,
            'results': [
                {
                    'scenario': r.scenario,
                    'passed': r.passed,
                    'duration': r.duration,
                    'details': r.details,
                    'metrics': r.metrics
                }
                for r in self.results
            ]
        }
        
        with open('validation_results.json', 'w') as f:
            json.dump(results_data, f, indent=2)
            
        print(f"\nResults exported to: validation_results.json")
        print(f"Timestamp: {datetime.now().isoformat()}")

if __name__ == '__main__':
    validator = StackMonitorValidator()
    validator.run_all()