package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	pb "stackmonitor.com/logproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Agent struct {
	id            string
	ingestionURL  string
	batchID       int64
	buffer        []*pb.LogEntry
	sampleRate    float64
	logPaths      []string
	filePositions map[string]int64
}

func NewAgent(id, ingestionURL string, logPaths []string) *Agent {
	positions := make(map[string]int64)
	for _, path := range logPaths {
		positions[path] = 0
	}
	
	return &Agent{
		id:            id,
		ingestionURL:  ingestionURL,
		batchID:       0,
		buffer:        make([]*pb.LogEntry, 0, 100),
		sampleRate:    0.3, // Sample 30% of logs
		logPaths:      logPaths,
		filePositions: positions,
	}
}

// Parse log line: [timestamp] [level] [type] message
var logPattern = regexp.MustCompile(`^\[(.*?)\]\s+\[(.*?)\]\s+\[(.*?)\]\s+(.*)$`)

func (a *Agent) parseLogLine(line, source string) *pb.LogEntry {
	matches := logPattern.FindStringSubmatch(line)
	if len(matches) != 5 {
		// Fallback for unparseable lines
		return &pb.LogEntry{
			TimestampNs: time.Now().UnixNano(),
			Level:       "INFO",
			Message:     line,
			Source:      source,
			AgentId:     a.id,
			Fields:      map[string]string{},
		}
	}
	
	timestamp, level, logType, message := matches[1], matches[2], matches[3], matches[4]
	
	// Parse timestamp
	t, err := time.Parse(time.RFC3339, strings.ReplaceAll(timestamp, "Z", "+00:00"))
	if err != nil {
		t = time.Now()
	}
	
	return &pb.LogEntry{
		TimestampNs: t.UnixNano(),
		Level:       level,
		Message:     message,
		Source:      source,
		AgentId:     a.id,
		Fields: map[string]string{
			"log_type": logType,
			"host":     "go-agent-host",
			"env":      "production",
		},
	}
}

func (a *Agent) tailLogs() {
	for {
		for _, logPath := range a.logPaths {
			// Check if file exists
			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				continue
			}
			
			file, err := os.Open(logPath)
			if err != nil {
				log.Printf("Error opening %s: %v", logPath, err)
				continue
			}
			
			// Seek to last known position
			_, err = file.Seek(a.filePositions[logPath], 0)
			if err != nil {
				file.Close()
				continue
			}
			
			scanner := bufio.NewScanner(file)
			linesRead := 0
			
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}
				
				// Sample logs based on sample rate
				if rand.Float64() > a.sampleRate {
					continue
				}
				
				// Parse and add to buffer
				source := strings.TrimSuffix(strings.Split(logPath, "/")[len(strings.Split(logPath, "/"))-1], ".log")
				entry := a.parseLogLine(line, source)
				a.buffer = append(a.buffer, entry)
				linesRead++
			}
			
			// Update file position
			newPos, _ := file.Seek(0, 1) // Get current position
			a.filePositions[logPath] = newPos
			file.Close()
			
			// Send batch if buffer is large enough
			if len(a.buffer) >= 10 {
				a.sendBatch()
			}
		}
		
		time.Sleep(1 * time.Second)
	}
}

func (a *Agent) sendBatch() {
	if len(a.buffer) == 0 {
		return
	}
	
	conn, err := grpc.Dial(a.ingestionURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}
	defer conn.Close()
	
	client := pb.NewLogIngestionClient(conn)
	stream, err := client.StreamLogs(context.Background())
	if err != nil {
		log.Printf("Failed to create stream: %v", err)
		return
	}
	
	a.batchID++
	
	// Calculate batch statistics
	errorCount := 0
	warnCount := 0
	for _, entry := range a.buffer {
		if entry.Level == "ERROR" {
			errorCount++
		} else if entry.Level == "WARN" {
			warnCount++
		}
	}
	
	batch := &pb.LogBatch{
		AgentId:     a.id,
		BatchId:     a.batchID,
		TimestampMs: time.Now().UnixMilli(),
		Logs:        a.buffer,
		Compression: pb.CompressionType_NONE,
		Metadata: map[string]string{
			"compression_ratio": "1.0",
			"sample_rate":       fmt.Sprintf("%.2f", a.sampleRate),
		},
	}
	
	if err := stream.Send(batch); err != nil {
		log.Printf("Failed to send batch: %v", err)
		return
	}
	
	ack, err := stream.Recv()
	if err != nil {
		log.Printf("Failed to receive ack: %v", err)
		return
	}
	
	if ack.Status == pb.AckStatus_SUCCESS {
		log.Printf("✅ Batch %d sent: %d logs (errors: %d, warns: %d, sample_rate: %.0f%%)",
			a.batchID, len(a.buffer), errorCount, warnCount, a.sampleRate*100)
		a.buffer = make([]*pb.LogEntry, 0, 100)
	} else {
		log.Printf("❌ Batch %d failed: %s", a.batchID, ack.Message)
	}
	
	stream.CloseSend()
}

func main() {
	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		agentID = "go-agent-1"
	}
	
	ingestionURL := os.Getenv("INGESTION_URL")
	if ingestionURL == "" {
		ingestionURL = "ingestion-service:50051"
	}
	
	// Log paths to monitor
	logPaths := []string{
		"/logs/tomcat.log",
		"/logs/nginx.log",
		"/logs/application.log",
	}
	
	agent := NewAgent(agentID, ingestionURL, logPaths)
	
	log.Printf("🚀 Go agent %s starting...", agentID)
	log.Printf("📁 Monitoring logs: %v", logPaths)
	log.Printf("🎯 Sample rate: %.0f%%", agent.sampleRate*100)
	log.Printf("📡 Ingestion URL: %s", ingestionURL)
	
	agent.tailLogs()
}
