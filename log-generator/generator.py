import time
import random
import os
import socket
from datetime import datetime

LOG_DIR = "/logs"
LOG_FILES = {
    "application": os.path.join(LOG_DIR, "application.log"),
    "tomcat": os.path.join(LOG_DIR, "tomcat.log"),
    "nginx": os.path.join(LOG_DIR, "nginx.log"),
}

SERVICES = ["payment-service", "user-service", "api-gateway"]
IP_ADDRESSES = ["192.168.1.100", "192.168.1.101", "10.0.0.50", "10.0.0.51"]
USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
]

HTTP_METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"]
HTTP_STATUS_CODES = {
    200: 0.70,
    201: 0.05,
    400: 0.10,
    404: 0.08,
    500: 0.05,
    503: 0.02,
}
HTTP_PATHS = ["/api/users", "/api/payments", "/api/orders", "/health", "/metrics", "/login", "/logout"]

# Log level distribution (configurable via environment variables)
# Default: INFO=80%, WARN=15%, ERROR=5%
LEVELS = {
    "INFO": float(os.getenv("LOG_RATE_INFO", "0.80")),
    "WARN": float(os.getenv("LOG_RATE_WARN", "0.15")),
    "ERROR": float(os.getenv("LOG_RATE_ERROR", "0.05"))
}

# Validate that rates sum to 1.0
total_rate = sum(LEVELS.values())
if abs(total_rate - 1.0) > 0.01:
    print(f"Warning: Log level rates sum to {total_rate}, adjusting to 1.0")
    # Normalize to sum to 1.0
    for level in LEVELS:
        LEVELS[level] = LEVELS[level] / total_rate

print(f"Log level distribution: INFO={LEVELS['INFO']*100:.1f}%, WARN={LEVELS['WARN']*100:.1f}%, ERROR={LEVELS['ERROR']*100:.1f}%")

APP_MESSAGES = {
    "INFO": [
        "Transaction processed: txn_id={id}, amount={amount:.2f}, latency={latency}ms",
        "User login successful: user_id={id}",
        "Cache hit for key: {key}",
    ],
    "WARN": [
        "Cache miss for key: {key}",
        "Request latency high: {latency}ms",
        "Rate limit approaching threshold: {threshold}%",
    ],
    "ERROR": [
        "Database timeout: host=db-replica-2, query=SELECT, timeout=5000ms",
        "OutOfMemory exception: Failed to allocate buffer",
        "NullPointerException at com.stackmonitor.UserService:{line}",
    ]
}

TOMCAT_MESSAGES = {
    "INFO": [
        "Server startup in {time}ms",
        "Deploying web application directory {path}",
        "Starting ProtocolHandler [\"http-nio-8080\"]",
    ],
    "WARN": [
        "Setting property 'source' to 'javax.xml.transform' did not find a matching property",
        "The web application [ROOT] appears to have started a thread named [Timer-0]",
    ],
    "ERROR": [
        "SEVERE: Error starting endpoint",
        "org.apache.catalina.core.StandardContext.startInternal Context [/app] startup failed",
        "java.sql.SQLException: Connection refused",
        "OutOfMemoryError: Java heap space",
    ]
}

def generate_application_log():
    """Generate application-style log (structured JSON-like)"""
    level = random.choices(list(LEVELS.keys()), weights=list(LEVELS.values()), k=1)[0]
    service = random.choice(SERVICES)
    msg_template = random.choice(APP_MESSAGES[level])
    
    msg = msg_template.format(
        id=random.randint(10000, 99999),
        amount=random.uniform(5.0, 500.0),
        latency=random.randint(50, 1500),
        key=f"user:profile:{random.randint(1000, 2000)}",
        line=random.randint(42, 300),
        threshold=random.randint(70, 95)
    )
    
    # Add intentional duplicates for Scenario 3
    if random.random() < 0.2:
        msg = "Database timeout: host=db-replica-2, query=SELECT, timeout=5000ms"
        level = "ERROR"
    
    timestamp = datetime.utcnow().isoformat() + "Z"
    return f"{timestamp} {level} [{service}] {msg}\n"

def generate_tomcat_log():
    """Generate Tomcat server log format"""
    level = random.choices(list(LEVELS.keys()), weights=list(LEVELS.values()), k=1)[0]
    msg_template = random.choice(TOMCAT_MESSAGES.get(level, TOMCAT_MESSAGES["INFO"]))
    
    # Tomcat format: DATE SEVERE [thread] message
    timestamp = datetime.now().strftime("%d-%b-%Y %H:%M:%S.%f")[:-3]
    thread = f"http-nio-8080-exec-{random.randint(1, 50)}"
    
    msg = msg_template.format(
        time=random.randint(2000, 5000),
        path=random.choice(["/opt/tomcat/webapps/ROOT", "/app", "/api"])
    )
    
    # Add stack trace for errors
    if level == "ERROR" and random.random() < 0.3:
        msg += "\n\tat org.apache.catalina.core.ApplicationFilterChain.doFilter(ApplicationFilterChain.java:227)"
        msg += "\n\tat org.apache.catalina.core.ApplicationFilterChain.internalDoFilter(ApplicationFilterChain.java:189)"
        msg += "\n\tat org.apache.catalina.core.ApplicationDispatcher.invoke(ApplicationDispatcher.java:646)"
    
    if level == "ERROR":
        level_str = "SEVERE"
    elif level == "WARN":
        level_str = "WARNING"
    else:
        level_str = "INFO"
    
    return f"{timestamp} {level_str} [{thread}] {msg}\n"

def generate_nginx_log():
    """Generate Nginx access log (Combined format)"""
    client_ip = random.choice(IP_ADDRESSES)
    timestamp = datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")
    method = random.choice(HTTP_METHODS)
    path = random.choice(HTTP_PATHS)
    protocol = "HTTP/1.1"
    status = random.choices(list(HTTP_STATUS_CODES.keys()), 
                        weights=list(HTTP_STATUS_CODES.values()), k=1)[0]
    body_bytes = random.randint(100, 50000)
    referer = random.choice(["https://example.com", "-", "https://stackmonitor.com"])
    user_agent = random.choice(USER_AGENTS)
    
    # Nginx Combined Log Format:
    # $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
    return f'{client_ip} - - [{timestamp}] "{method} {path} {protocol}" {status} {body_bytes} "{referer}" "{user_agent}"\n'

def get_log_line(log_type):
    """Get a log line based on type"""
    if log_type == "application":
        return generate_application_log()
    elif log_type == "tomcat":
        return generate_tomcat_log()
    elif log_type == "nginx":
        return generate_nginx_log()
    else:
        return generate_application_log()

if __name__ == "__main__":
    print(f"Log generators starting. Writing to {LOG_DIR}...")
    
    # Ensure log directory exists
    os.makedirs(LOG_DIR, exist_ok=True)
    
    # Open file handles for each log type
    file_handles = {}
    for log_type, filepath in LOG_FILES.items():
        file_handles[log_type] = open(filepath, "a", encoding="utf-8")
        print(f"Writing {log_type} logs to {filepath}")
    
    try:
        while True:
            for log_type, fh in file_handles.items():
                line = get_log_line(log_type)
                fh.write(line)
                fh.flush()
            
            # Vary generation rate: 0.1-1.0 seconds between batches
            time.sleep(random.uniform(0.1, 1.0))
            
    except KeyboardInterrupt:
        print("Stopping generators...")
    finally:
        for fh in file_handles.values():
            fh.close()
