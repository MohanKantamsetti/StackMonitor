package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var chConn clickhouse.Conn

func main() {
	chAddr := os.Getenv("CLICKHOUSE_ADDR")
	if chAddr == "" {
		chAddr = "clickhouse:9000"
	}

	var err error
	chConn, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{chAddr},
	})
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chConn.Close()

	r := gin.Default()
	r.Use(corsMiddleware())

	r.GET("/api/v1/logs", getLogs)
	r.GET("/api/v1/logs/stats", getLogsStats)
	r.GET("/api/v1/metrics/error-rate", getErrorRate)
	r.GET("/api/v1/logs/stream", streamLogs)

	log.Println("API Server starting on :5000")
	r.Run(":5000")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source"`
	AgentID   string `json:"agent_id"`
}

func getLogs(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	level := c.Query("level")
	service := c.Query("service")

	query := "SELECT timestamp, level, message, source, agent_id FROM logs WHERE 1=1"
	if level != "" {
		query += fmt.Sprintf(" AND level = '%s'", level)
	}
	if service != "" {
		query += fmt.Sprintf(" AND service = '%s'", service)
	}
	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %s", limit)

	rows, err := chConn.Query(context.Background(), query)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	logs := []LogEntry{}
	for rows.Next() {
		var entry LogEntry
		var ts time.Time
		if err := rows.Scan(&ts, &entry.Level, &entry.Message, &entry.Source, &entry.AgentID); err != nil {
			continue
		}
		entry.Timestamp = ts.Format(time.RFC3339)
		logs = append(logs, entry)
	}

	c.JSON(200, logs)
}

func getLogsStats(c *gin.Context) {
	// Get total count and count by level
	query := `
		SELECT 
			count() as total,
			countIf(level = 'ERROR') as errors,
			countIf(level = 'WARN') as warns,
			countIf(level = 'INFO') as infos
		FROM logs
	`
	
	var total, errors, warns, infos uint64
	err := chConn.QueryRow(context.Background(), query).Scan(&total, &errors, &warns, &infos)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{
		"total":  total,
		"errors": errors,
		"warns":  warns,
		"infos":  infos,
	})
}

func getErrorRate(c *gin.Context) {
	query := `
		SELECT 
			toStartOfMinute(timestamp) as minute,
			countIf(level = 'ERROR') as errors,
			count() as total
		FROM logs
		WHERE timestamp > now() - INTERVAL 1 HOUR
		GROUP BY minute
		ORDER BY minute
	`

	rows, err := chConn.Query(context.Background(), query)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Metric struct {
		Minute string  `json:"minute"`
		Errors int     `json:"errors"`
		Total  int     `json:"total"`
		Rate   float64 `json:"rate"`
	}

	metrics := []Metric{}
	for rows.Next() {
		var m Metric
		var minute time.Time
		if err := rows.Scan(&minute, &m.Errors, &m.Total); err != nil {
			continue
		}
		m.Minute = minute.Format(time.RFC3339)
		if m.Total > 0 {
			m.Rate = float64(m.Errors) / float64(m.Total) * 100
		}
		metrics = append(metrics, m)
	}

	c.JSON(200, metrics)
}

func streamLogs(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		query := "SELECT timestamp, level, message, source, agent_id FROM logs ORDER BY timestamp DESC LIMIT 10"
		rows, err := chConn.Query(context.Background(), query)
		if err != nil {
			continue
		}

		logs := []LogEntry{}
		for rows.Next() {
			var entry LogEntry
			var ts time.Time
			if err := rows.Scan(&ts, &entry.Level, &entry.Message, &entry.Source, &entry.AgentID); err != nil {
				continue
			}
			entry.Timestamp = ts.Format(time.RFC3339)
			logs = append(logs, entry)
		}
		rows.Close()

		data, _ := json.Marshal(logs)
		if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}

