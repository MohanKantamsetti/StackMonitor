package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type APIServer struct {
	db driver.Conn
}

func setupRouter(api *APIServer) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	apiGroup := r.Group("/api/v1")
	{
		// GET /api/v1/logs
		apiGroup.GET("/logs", func(c *gin.Context) {
			service := c.Query("service")
			level := c.Query("level")
			limitStr := c.Query("limit")
			limit := 100
			if limitStr != "" {
				if l, err := strconv.Atoi(limitStr); err == nil {
					limit = l
				}
			}

			query := "SELECT timestamp, level, service, message, trace_id, agent_id FROM stackmonitor.logs WHERE 1=1"
			args := []interface{}{}

			if service != "" {
				query += " AND service = ?"
				args = append(args, service)
			}
			if level != "" {
				query += " AND level = ?"
				args = append(args, level)
			}

			query += " ORDER BY timestamp DESC LIMIT ?"
			args = append(args, limit)

			rows, err := api.db.Query(context.Background(), query, args...)
			if err != nil {
				log.Printf("Query error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()

			var logs []map[string]interface{}
			for rows.Next() {
				var timestamp time.Time
				var logLevel, service, message, traceID, agentID string

				if err := rows.Scan(&timestamp, &logLevel, &service, &message, &traceID, &agentID); err != nil {
					log.Printf("Error scanning row: %v", err)
					continue
				}

				logs = append(logs, map[string]interface{}{
					"timestamp": timestamp.Format(time.RFC3339),
					"level":     logLevel,
					"service":   service,
					"message":   message,
					"trace_id":  traceID,
					"agent_id":  agentID,
				})
			}

			// Ensure logs is never null
			if logs == nil {
				logs = []map[string]interface{}{}
			}

			result := gin.H{"logs": logs, "count": len(logs)}

			// Check if request wants HTML (from browser)
			if c.GetHeader("Accept") == "text/html" || c.Query("format") == "html" {
				c.Header("Content-Type", "text/html; charset=utf-8")
				renderLogsHTML(c, logs, level, service, limit)
				return
			}

			c.JSON(http.StatusOK, result)
		})

		// GET /api/v1/logs/stats
		apiGroup.GET("/logs/stats", func(c *gin.Context) {
			// Get log statistics
			var totalCount, errorCount, warnCount, infoCount uint64
			
			// Total count
			err := api.db.QueryRow(context.Background(), "SELECT count() FROM stackmonitor.logs").Scan(&totalCount)
			if err != nil {
				log.Printf("Error getting total count: %v", err)
			}
			
			// Error count
			err = api.db.QueryRow(context.Background(), "SELECT count() FROM stackmonitor.logs WHERE level = 'ERROR'").Scan(&errorCount)
			if err != nil {
				log.Printf("Error getting error count: %v", err)
			}
			
			// Warn count
			err = api.db.QueryRow(context.Background(), "SELECT count() FROM stackmonitor.logs WHERE level = 'WARN'").Scan(&warnCount)
			if err != nil {
				log.Printf("Error getting warn count: %v", err)
			}
			
			// Info count
			err = api.db.QueryRow(context.Background(), "SELECT count() FROM stackmonitor.logs WHERE level = 'INFO'").Scan(&infoCount)
			if err != nil {
				log.Printf("Error getting info count: %v", err)
			}
			
			c.JSON(http.StatusOK, gin.H{
				"total": totalCount,
				"errors": errorCount,
				"warnings": warnCount,
				"info": infoCount,
			})
		})

		// GET /api/v1/metrics/error-rate
		apiGroup.GET("/metrics/error-rate", func(c *gin.Context) {
			service := c.Query("service")
			rangeStr := c.Query("range")
			if rangeStr == "" {
				rangeStr = "1h"
			}

			var interval string
			var timeRange string
			switch rangeStr {
			case "15m":
				interval = "1 minute"
				timeRange = "15 MINUTE"
			case "1h":
				interval = "1 minute"
				timeRange = "1 HOUR"
			case "6h":
				interval = "5 minute"
				timeRange = "6 HOUR"
			case "24h":
				interval = "15 minute"
				timeRange = "24 HOUR"
			case "all":
				interval = "1 hour"
				timeRange = "30 DAY"
			default:
				interval = "1 minute"
				timeRange = "1 HOUR"
			}

			query := `
				SELECT 
					toStartOfInterval(timestamp, INTERVAL ` + interval + `) as time,
					count(*) as error_count
				FROM stackmonitor.logs
				WHERE level = 'ERROR'
			`
			args := []interface{}{}

			if service != "" {
				query += " AND service = ?"
				args = append(args, service)
			}

			query += " AND timestamp >= now() - INTERVAL " + timeRange + " GROUP BY time ORDER BY time"

			rows, err := api.db.Query(context.Background(), query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()

			var metrics []map[string]interface{}
			for rows.Next() {
				var timeVal time.Time
				var count uint64

				if err := rows.Scan(&timeVal, &count); err != nil {
					log.Printf("Error scanning row: %v", err)
					continue
				}

				metrics = append(metrics, map[string]interface{}{
					"time":  timeVal.Format(time.RFC3339),
					"count": count,
				})
			}

			c.JSON(http.StatusOK, gin.H{"metrics": metrics})
		})

		// POST /api/v1/query (Natural Language Query)
		apiGroup.POST("/query", func(c *gin.Context) {
			var req struct {
				Query string `json:"query"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}

			// Simple keyword-based query parsing
			query := req.Query
			results := make(map[string]interface{})

			if contains(query, "error", "errors") {
				// Get recent errors
				rows, err := api.db.Query(context.Background(),
					"SELECT service, count(*) as cnt FROM stackmonitor.logs WHERE level = 'ERROR' AND timestamp >= now() - INTERVAL 1 HOUR GROUP BY service",
				)
				if err == nil {
					defer rows.Close()
					var errorCounts []map[string]interface{}
					for rows.Next() {
						var service string
						var count uint64
						if err := rows.Scan(&service, &count); err == nil {
							errorCounts = append(errorCounts, map[string]interface{}{
								"service": service,
								"count":   count,
							})
						}
					}
					results["errors_by_service"] = errorCounts
				}
			}

			c.JSON(http.StatusOK, gin.H{"query": query, "results": results})
		})

		// WebSocket for live log stream
		apiGroup.GET("/logs/stream", func(c *gin.Context) {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				log.Printf("WebSocket upgrade failed: %v", err)
				return
			}
			defer conn.Close()

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			lastTimestamp := time.Now()

			for {
				select {
				case <-ticker.C:
					query := "SELECT timestamp, level, service, message, trace_id, agent_id FROM stackmonitor.logs WHERE timestamp > ? ORDER BY timestamp LIMIT 100"
					rows, err := api.db.Query(context.Background(), query, lastTimestamp)
					if err != nil {
						log.Printf("Query error: %v", err)
						continue
					}

					var logs []map[string]interface{}
					for rows.Next() {
						var timestamp time.Time
						var logLevel, service, message, traceID, agentID string

						if err := rows.Scan(&timestamp, &logLevel, &service, &message, &traceID, &agentID); err != nil {
							continue
						}

						if timestamp.After(lastTimestamp) {
							lastTimestamp = timestamp
						}

						logs = append(logs, map[string]interface{}{
							"timestamp": timestamp.Format(time.RFC3339),
							"level":     logLevel,
							"service":   service,
							"message":   message,
							"trace_id":  traceID,
							"agent_id":  agentID,
						})
					}
					rows.Close()

					if len(logs) > 0 {
						data, _ := json.Marshal(logs)
						if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
							log.Printf("WebSocket write error: %v", err)
							return
						}
					}
				}
			}
		})
	}

	return r
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// Render logs as HTML for browser viewing
func renderLogsHTML(c *gin.Context, logs []map[string]interface{}, level, service string, limit int) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>StackMonitor API - Logs</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            padding: 20px;
        }
        .container { max-width: 1400px; margin: 0 auto; }
        .header {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header h1 { color: #333; margin-bottom: 10px; }
        .filters {
            display: flex;
            gap: 15px;
            margin-top: 15px;
            flex-wrap: wrap;
        }
        .filter-badge {
            padding: 6px 12px;
            background: #e3f2fd;
            border-radius: 4px;
            font-size: 14px;
            color: #1976d2;
        }
        .log-table {
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        table { width: 100%; border-collapse: collapse; }
        thead {
            background: #f8f9fa;
            border-bottom: 2px solid #e0e0e0;
        }
        th {
            padding: 15px;
            text-align: left;
            font-weight: 600;
            color: #555;
            font-size: 13px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        td {
            padding: 12px 15px;
            border-bottom: 1px solid #f0f0f0;
            font-size: 14px;
        }
        tr:hover { background: #f8f9fa; }
        .level {
            padding: 4px 8px;
            border-radius: 4px;
            font-weight: 600;
            font-size: 12px;
            display: inline-block;
        }
        .level-ERROR { background: #ffebee; color: #c62828; }
        .level-WARN { background: #fff8e1; color: #f57c00; }
        .level-INFO { background: #e3f2fd; color: #1976d2; }
        .timestamp { color: #666; font-family: 'Courier New', monospace; font-size: 12px; }
        .service { color: #666; font-weight: 500; }
        .message { color: #333; font-family: 'Courier New', monospace; font-size: 13px; word-break: break-word; }
        .stats {
            background: white;
            padding: 15px 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .api-link {
            color: #1976d2;
            text-decoration: none;
            font-weight: 500;
        }
        .api-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 StackMonitor Logs</h1>
            <div class="filters">`

	// Add filter badges
	if level != "" {
		html += fmt.Sprintf(`<span class="filter-badge">Level: %s</span>`, level)
	}
	if service != "" {
		html += fmt.Sprintf(`<span class="filter-badge">Service: %s</span>`, service)
	}
	html += fmt.Sprintf(`<span class="filter-badge">Limit: %d</span>`, limit)

	html += `</div>
        </div>
        <div class="stats">
            <div><strong>Total Logs:</strong> ` + fmt.Sprintf("%d", len(logs)) + `</div>
            <div><a href="/api/v1/logs?format=json" class="api-link">📄 View JSON</a></div>
        </div>
        <div class="log-table">
            <table>
                <thead>
                    <tr>
                        <th style="width: 180px;">Timestamp</th>
                        <th style="width: 80px;">Level</th>
                        <th style="width: 150px;">Service</th>
                        <th>Message</th>
                    </tr>
                </thead>
                <tbody>`

	// Render log rows
	for _, logEntry := range logs {
		timestamp := logEntry["timestamp"]
		logLevel := logEntry["level"]
		logService := logEntry["service"]
		logMessage := logEntry["message"]

		html += fmt.Sprintf(`
                    <tr>
                        <td class="timestamp">%v</td>
                        <td><span class="level level-%v">%v</span></td>
                        <td class="service">%v</td>
                        <td class="message">%v</td>
                    </tr>`,
			timestamp, logLevel, logLevel, logService, logMessage)
	}

	html += `
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func main() {
	clickhouseAddr := os.Getenv("CLICKHOUSE_ADDR")
	if clickhouseAddr == "" {
		clickhouseAddr = "clickhouse:9000"
	}

	// ClickHouse connection - dev mode (no authentication)
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseAddr},
		Auth: clickhouse.Auth{
			Database: "stackmonitor",
			// No username/password for dev mode
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping ClickHouse: %v", err)
	}

	api := &APIServer{db: conn}
	r := setupRouter(api)
	r.Run(":5000")
}
