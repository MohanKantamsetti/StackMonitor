#!/usr/bin/env python3
import os
import time
import random
from datetime import datetime

# Configuration from environment variables
ERROR_RATE = float(os.getenv('ERROR_RATE', '0.15'))  # 15% errors by default
WARN_RATE = float(os.getenv('WARN_RATE', '0.25'))    # 25% warnings by default
LOG_RATE = float(os.getenv('LOG_RATE', '0.5'))       # Logs per second (0.5 = 2 logs/sec)
DEBUG_MODE = os.getenv('DEBUG_MODE', 'false').lower() == 'true'

LOG_TYPES = ['Tomcat', 'Nginx', 'Application']

# More realistic and detailed messages
MESSAGES = {
    'Tomcat': {
        'INFO': [
            'HTTP request processed: GET /api/users - 200 OK (25ms)',
            'HTTP request processed: POST /api/orders - 201 Created (143ms)',
            'Session created for user: user_{id} from IP 192.168.1.{ip}',
            'Deployed application context [/myapp] successfully',
            'Started processing batch job: invoice-generator',
            'Cache warm-up completed: 10000 entries loaded',
        ],
        'WARN': [
            'Slow query detected: SELECT * FROM orders took {ms}ms',
            'Connection pool at 80% capacity ({count}/100 connections)',
            'Session timeout for user_{id} after 30 minutes of inactivity',
            'Retry attempt {count}/3 for payment gateway',
            'Memory usage high: {percent}% of heap used',
            'Deprecated API endpoint accessed: /api/v1/legacy',
        ],
        'ERROR': [
            'Database connection failed: Connection timeout after 30s',
            'NullPointerException in OrderService.processPayment() line 245',
            'OutOfMemoryError: Java heap space - unable to allocate {size}MB',
            'Failed to process order #{id}: Payment gateway returned 500',
            'Session replication failed: Unable to reach cluster node-{node}',
            'ClassNotFoundException: com.example.legacy.OldPaymentProcessor',
        ],
        'DEBUG': [
            'Request parameters: userId={id}, action=update',
            'Database query: SELECT * FROM users WHERE id = {id}',
            'Cache hit for key: user_profile_{id}',
            'Response serialization completed in {ms}ms',
        ]
    },
    'Nginx': {
        'INFO': [
            '192.168.1.{ip} - GET /index.html HTTP/1.1 200 ({size} bytes)',
            '192.168.1.{ip} - POST /api/login HTTP/1.1 201 (auth successful)',
            'SSL handshake completed successfully for client 192.168.1.{ip}',
            'Cache hit for /static/bundle.js - served from memory',
            'Upstream server backend-{node} responded in {ms}ms',
        ],
        'WARN': [
            'Rate limit exceeded for IP 192.168.1.{ip} - 100 requests in 60s',
            'Slow upstream response from backend-{node}: {ms}ms',
            '192.168.1.{ip} - GET /api/data HTTP/1.1 429 Too Many Requests',
            'Client certificate expiring in {days} days for domain example.com',
            'Worker process {pid} exited on signal 11',
        ],
        'ERROR': [
            'Upstream server connection failed: Connection refused to backend-{node}',
            '192.168.1.{ip} - GET /api/users HTTP/1.1 502 Bad Gateway',
            'SSL certificate verification failed for backend-{node}',
            'Failed to allocate {size}KB from shared memory zone "cache"',
            '192.168.1.{ip} - POST /api/upload HTTP/1.1 413 Request Entity Too Large',
            'Upstream timed out ({ms}ms) while connecting to backend-{node}',
        ],
        'DEBUG': [
            'Request headers: User-Agent: Mozilla/5.0, Accept: application/json',
            'Proxy pass to upstream: http://backend-{node}:8080',
            'Cache status: MISS for /api/products?page={page}',
            'Connection keepalive timeout: 75s',
        ]
    },
    'Application': {
        'INFO': [
            'User authentication successful for user_{id}',
            'Payment processing initiated: order #{id}, amount ${amount}',
            'Database query executed successfully in {ms}ms',
            'Email sent to user_{id}: order confirmation',
            'Background job completed: email_sender ({count} emails)',
            'API request: /users/{id} - completed in {ms}ms',
        ],
        'WARN': [
            'Cache miss for key: user_profile_{id} - fetching from database',
            'Retry attempt {count}/3 for external API call',
            'Database connection pool nearing capacity: {percent}%',
            'Slow query detected: {ms}ms for complex report generation',
            'External API latency high: {ms}ms for payment gateway',
            'Deprecated function called: old_payment_processor()',
        ],
        'ERROR': [
            'Payment failed: Insufficient funds for order #{id}',
            'Database deadlock detected in transaction processing',
            'Redis connection failed: ECONNREFUSED 127.0.0.1:6379',
            'Unhandled exception in UserService: ValueError at line {line}',
            'Message queue error: RabbitMQ connection lost',
            'S3 upload failed: AccessDenied for bucket user-uploads',
            'Elasticsearch query failed: CircuitBreaker[parent] triggered',
        ],
        'DEBUG': [
            'Cache TTL set: user_profile_{id} expires in 3600s',
            'SQL query: SELECT * FROM orders WHERE user_id = {id}',
            'Serializing response: {size} KB JSON payload',
            'HTTP client request: GET https://api.external.com/data',
        ]
    }
}

def get_log_level():
    """Determine log level based on configured error/warn rates"""
    rand = random.random()
    if rand < ERROR_RATE:
        return 'ERROR'
    elif rand < ERROR_RATE + WARN_RATE:
        return 'WARN'
    elif rand < ERROR_RATE + WARN_RATE + 0.15:  # 15% DEBUG
        return 'DEBUG'
    else:
        return 'INFO'

def generate_log():
    log_type = random.choice(LOG_TYPES)
    level = get_log_level()
    
    # Get appropriate message for type and level
    message_template = random.choice(MESSAGES[log_type][level])
    
    # Fill in template variables
    message = message_template.format(
        id=random.randint(1000, 9999),
        ip=random.randint(1, 255),
        ms=random.randint(10, 500),
        size=random.randint(100, 5000),
        count=random.randint(1, 10),
        percent=random.randint(70, 95),
        node=random.randint(1, 5),
        days=random.randint(1, 30),
        pid=random.randint(1000, 9999),
        page=random.randint(1, 50),
        amount=random.randint(10, 1000),
        line=random.randint(100, 500)
    )
    
    timestamp = datetime.utcnow().isoformat()
    log_entry = f"[{timestamp}] [{level}] [{log_type}] {message}"
    
    return log_entry, log_type

def main():
    log_dir = '/logs'
    os.makedirs(log_dir, exist_ok=True)
    
    log_files = {
        'Tomcat': open(os.path.join(log_dir, 'tomcat.log'), 'a', buffering=1),
        'Nginx': open(os.path.join(log_dir, 'nginx.log'), 'a', buffering=1),
        'Application': open(os.path.join(log_dir, 'application.log'), 'a', buffering=1),
    }
    
    print(f"Log generator started with configuration:")
    print(f"  ERROR_RATE: {ERROR_RATE * 100}%")
    print(f"  WARN_RATE: {WARN_RATE * 100}%")
    print(f"  LOG_RATE: {LOG_RATE} logs/second")
    print(f"  DEBUG_MODE: {DEBUG_MODE}")
    print(f"Writing logs to {log_dir}/")
    
    try:
        while True:
            log_entry, log_type = generate_log()
            log_files[log_type].write(log_entry + '\n')
            
            if DEBUG_MODE:
                print(log_entry)
            
            # Sleep based on configured log rate
            time.sleep(1.0 / LOG_RATE if LOG_RATE > 0 else 0.5)
    
    except KeyboardInterrupt:
        print("\nShutting down log generator...")
    finally:
        for f in log_files.values():
            f.close()

if __name__ == '__main__':
    main()
