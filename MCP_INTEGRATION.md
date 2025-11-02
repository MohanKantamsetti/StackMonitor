# StackMonitor MCP Server Integration with Cursor

## Overview
The StackMonitor MCP (Model Context Protocol) server provides a natural language interface to query and analyze system logs. It's now fully functional and ready to be integrated with Cursor's Agent Mode.

## Current Status
✅ **MCP Server Running**: http://localhost:5001  
✅ **Natural Language Processing**: Fully functional  
✅ **API Integration**: Connected to ClickHouse database  
✅ **Response Generation**: Enhanced with descriptive summaries  

## Available Endpoints

### POST /mcp/query
Send natural language queries about your logs.

**Request:**
```json
{
  "query": "What are the recent errors?"
}
```

**Response:**
```json
{
  "response": "I found 50 error logs",
  "data": [/* array of log entries */]
}
```

## Example Queries

### 1. Error Analysis
- "What are the recent errors?"
- "Show me all error logs"
- "How many errors occurred?"

### 2. Trend Analysis
- "Show me error rate trends"
- "What's the error rate for the last hour?"

### 3. Log Description
- "Describe my logs"
- "Give me a summary of recent logs"
- "What's the status of the system?"

### 4. Service-Specific
- "Show me nginx logs"
- "What errors are tomcat reporting?"
- "Application logs from the last hour"

## Testing the MCP Server

### Using PowerShell:
```powershell
$body = @{query="What are the recent errors?"} | ConvertTo-Json
curl http://localhost:5001/mcp/query -Method POST -Body $body -ContentType "application/json"
```

### Using curl (Linux/Mac):
```bash
curl -X POST http://localhost:5001/mcp/query \
  -H "Content-Type: application/json" \
  -d '{"query":"What are the recent errors?"}'
```

## Integration with Cursor

### Manual Testing in Cursor Agent Mode

1. **Open Cursor Settings**
   - Press `Ctrl+Shift+P` (Windows) or `Cmd+Shift+P` (Mac)
   - Type "Open User Settings (JSON)"

2. **Add MCP Server Configuration**
   Add this to your Cursor settings:
   ```json
   {
     "cursor.mcp.servers": [
       {
         "name": "StackMonitor",
         "url": "http://localhost:5001/mcp/query",
         "method": "POST",
         "description": "Query StackMonitor logs"
       }
     ]
   }
   ```

3. **Use in Agent Mode**
   - Open Cursor Agent Mode (Cmd/Ctrl + K)
   - Type: "@stackmonitor What are the recent errors?"
   - The agent will query the MCP server and return results

## Sample Responses

### Query: "What are the recent errors?"
**Response:**
```
I found 50 error logs
```
**Data:** Returns 50 most recent ERROR-level log entries

### Query: "describe my logs"
**Response:**
```
I found 100 logs: 23 errors, 35 warnings, 42 info
```
**Data:** Returns 100 recent logs with distribution

### Query: "Show me error rate trends"
**Response:**
```
Error rate analysis for the last hour: 22.5% error rate (45 errors out of 200 logs)
```
**Data:** Returns minute-by-minute error rate metrics

## Architecture

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Cursor    │  HTTP   │  MCP Server │  HTTP   │ API Server  │
│ Agent Mode  │────────>│   :5001     │────────>│   :5000     │
└─────────────┘         └─────────────┘         └─────────────┘
                              │                         │
                              │                         │
                              v                         v
                        ┌─────────────────────────────────┐
                        │     ClickHouse Database         │
                        │    (Logs Storage: 2000+ logs)   │
                        └─────────────────────────────────┘
```

## Current System Metrics
- **Total Logs**: ~2000
- **Error Rate**: ~20%
- **Warning Rate**: ~22%
- **Services Monitored**: Tomcat, Nginx, Application
- **Agents**: Go Agent (active), Python Agent (pending)

## Troubleshooting

### MCP Server Not Responding
```bash
# Check if server is running
docker logs stackmonitor-poc-mcp-server-1 --tail 20

# Restart if needed
docker compose restart mcp-server
```

### No Data Returned
- Check that logs are being generated: `docker logs stackmonitor-poc-log-generator-1`
- Verify ClickHouse has data: `docker exec stackmonitor-poc-clickhouse-1 clickhouse-client --query "SELECT count() FROM logs"`

## Next Steps

1. **Test in Cursor Agent Mode**: Try the example queries
2. **Customize Queries**: Create domain-specific queries for your use case
3. **Add LLM Integration**: Configure Gemini API key for smarter query processing
4. **Extend Capabilities**: Add more tools (log correlation, anomaly detection, etc.)

---

**Status**: ✅ Fully Operational  
**Last Updated**: 2025-11-02  
**Maintainer**: StackMonitor Team

