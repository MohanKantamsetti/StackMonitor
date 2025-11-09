package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	pb "stackmonitor.com/ingestion-service/proto/logproto"
)

var (
	port          = ":50051"
	clickhouseAddr = "clickhouse:9000"
	database     = "stackmonitor"
	batchSize    = 100 // Number of logs to buffer before insert
	batchTimeout = 5 * time.Second
)

type ingestionServer struct {
	pb.UnimplementedLogIngestionServer
	db         driver.Conn
	logChan    chan *pb.LogEntry
	dedupCache *sync.Map // PoC deduplication
	encoder    *zstd.Encoder
	decoder    *zstd.Decoder
	
	// Metrics
	batchesReceived   atomic.Uint64
	logsReceived      atomic.Uint64
	logsProcessed     atomic.Uint64
	logsDuplicate     atomic.Uint64
	logsInserted      atomic.Uint64
	insertsFailed     atomic.Uint64
	bytesReceived     atomic.Uint64
	bytesDecompressed atomic.Uint64
	startTime         time.Time
	lastInsertTime    atomic.Int64
}

// Deduplication: in-memory hash cache with automatic expiration
// Detects duplicate log messages within a 60-second window
func (s *ingestionServer) isDuplicate(entry *pb.LogEntry) bool {
	// Hash based on message content, level, and service (NOT timestamp)
	// This catches the same error/warning occurring multiple times within 60s
	service := entry.Fields["service"]
	if service == "" {
		service = "unknown"
	}
	
	// Create hash from: message + level + service
	// Do NOT include timestamp - we want to catch duplicate messages even if timestamps differ
	hash := fmt.Sprintf("%s-%s-%s", entry.Message, entry.Level, service)
	
	if _, loaded := s.dedupCache.LoadOrStore(hash, true); loaded {
		return true // Duplicate found
	}
	
	// Expire cache entries after 60s to prevent memory leak
	// After 60s, the same error can be logged again (not considered a duplicate anymore)
	time.AfterFunc(60*time.Second, func() { s.dedupCache.Delete(hash) })
	return false
}

// gRPC StreamLogs implementation
func (s *ingestionServer) StreamLogs(stream pb.LogIngestion_StreamLogsServer) error {
	for {
		batch, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		s.batchesReceived.Add(1)
		s.logsReceived.Add(uint64(len(batch.Logs)))

		// Use logs directly from batch
		logsToProcess := batch.Logs
		
		// Handle compression if enabled
		if batch.Compression == pb.CompressionType_ZSTD && len(batch.CompressedPayload) > 0 {
			s.bytesReceived.Add(uint64(len(batch.CompressedPayload)))
			log.Printf("Received compressed batch %d (%d bytes compressed, original: %d bytes)", 
				batch.BatchId, len(batch.CompressedPayload), batch.OriginalSize)
			
			// Decompress payload
			decompressed, err := s.decoder.DecodeAll(batch.CompressedPayload, nil)
			if err != nil {
				log.Printf("Failed to decompress: %v", err)
				stream.Send(&pb.Ack{
					BatchId:           batch.BatchId,
					Status:            pb.AckStatus_RETRY,
					Message:           fmt.Sprintf("Decompression failed: %v", err),
					ServerTimestampMs: time.Now().UnixMilli(),
				})
				continue
			}
			s.bytesDecompressed.Add(uint64(len(decompressed)))
			
			// Parse decompressed payload into logs
			// The decompressed data is a concatenation of serialized LogEntry messages
			// Since we don't have delimiters, we'll use the batch.Logs as reference
			// and just update metrics - actual logs are already in batch.Logs
			// In production, you'd want to implement proper framing or use the decompressed data
			logsToProcess = batch.Logs
			
			log.Printf("Decompressed batch %d: %d logs from %d bytes", 
				batch.BatchId, len(logsToProcess), len(decompressed))
		} else if len(batch.Logs) > 0 {
			// Track uncompressed bytes (estimate)
			for _, entry := range batch.Logs {
				entrySize, _ := proto.Marshal(entry)
				s.bytesReceived.Add(uint64(len(entrySize)))
			}
		}

		processedCount := 0
		duplicateCount := 0
		for _, entry := range logsToProcess {
			// Apply deduplication
			if !s.isDuplicate(entry) {
				s.logChan <- entry
				processedCount++
				s.logsProcessed.Add(1)
			} else {
				duplicateCount++
				s.logsDuplicate.Add(1)
			}
		}
		log.Printf("📥 Received batch %d: %d logs (processed: %d, duplicates: %d)", 
			batch.BatchId, len(logsToProcess), processedCount, duplicateCount)

		if err := stream.Send(&pb.Ack{
			BatchId:           batch.BatchId,
			Status:            pb.AckStatus_SUCCESS,
			Message:           fmt.Sprintf("Processed %d/%d logs", processedCount, len(logsToProcess)),
			ServerTimestampMs: time.Now().UnixMilli(),
		}); err != nil {
			return err
		}
	}
}

// Batch writer for ClickHouse
func (s *ingestionServer) batchWriter() {
	ticker := time.NewTicker(batchTimeout)
	defer ticker.Stop()
	buffer := make([]*pb.LogEntry, 0, batchSize)

	for {
		select {
		case entry := <-s.logChan:
			buffer = append(buffer, entry)
			if len(buffer) >= batchSize {
				s.insertBatch(buffer)
				buffer = make([]*pb.LogEntry, 0, batchSize)
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				s.insertBatch(buffer)
				buffer = make([]*pb.LogEntry, 0, batchSize)
			}
		}
	}
}

func (s *ingestionServer) insertBatch(logs []*pb.LogEntry) {
	ctx := context.Background()
	batch, err := s.db.PrepareBatch(ctx, "INSERT INTO stackmonitor.logs")
	if err != nil {
		log.Printf("Failed to prepare batch: %v", err)
		return
	}
	defer batch.Abort()

	for _, entry := range logs {
		service := entry.Fields["service"]
		if service == "" {
			service = "unknown"
		}
		traceID := entry.Fields["trace_id"]
		agentID := entry.AgentId

		err := batch.Append(
			time.Unix(0, entry.TimestampNs),
			entry.Level,
			service,
			entry.Message,
			traceID,
			agentID,
			entry.Fields, // Using fields as metadata for PoC
		)
		if err != nil {
			log.Printf("Failed to append to batch: %v", err)
			return
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("❌ Failed to send batch: %v", err)
		s.insertsFailed.Add(1)
		return
	}
	s.logsInserted.Add(uint64(len(logs)))
	s.lastInsertTime.Store(time.Now().Unix())
	log.Printf("✅ Inserted %d logs into ClickHouse", len(logs))
}

// HTTP handler for health checks
func (s *ingestionServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	lastInsert := time.Unix(s.lastInsertTime.Load(), 0)
	timeSinceLast := time.Since(lastInsert)
	healthy := timeSinceLast < 2*time.Minute
	
	status := "healthy"
	statusCode := http.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}
	
	response := map[string]interface{}{
		"status":                status,
		"uptime_seconds":        time.Since(s.startTime).Seconds(),
		"last_insert_ago":       timeSinceLast.Seconds(),
		"log_chan_size":         len(s.logChan),
		"log_chan_capacity":     cap(s.logChan),
		"clickhouse_connected":  s.db != nil,
	}
	
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// HTTP handler for metrics
func (s *ingestionServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	uptime := time.Since(s.startTime).Seconds()
	bytesReceived := s.bytesReceived.Load()
	bytesDecompressed := s.bytesDecompressed.Load()
	
	compressionRatio := 1.0
	if bytesReceived > 0 && bytesDecompressed > 0 {
		compressionRatio = float64(bytesDecompressed) / float64(bytesReceived)
	}
	
	logsProcessed := s.logsProcessed.Load()
	logsInserted := s.logsInserted.Load()
	
	response := map[string]interface{}{
		"uptime_seconds":       uptime,
		"batches_received":     s.batchesReceived.Load(),
		"logs_received":        s.logsReceived.Load(),
		"logs_processed":       logsProcessed,
		"logs_duplicate":       s.logsDuplicate.Load(),
		"logs_inserted":        logsInserted,
		"inserts_failed":       s.insertsFailed.Load(),
		"bytes_received":       bytesReceived,
		"bytes_decompressed":   bytesDecompressed,
		"compression_ratio":    compressionRatio,
		"logs_per_second":      float64(logsProcessed) / uptime,
		"insert_rate":          float64(logsInserted) / uptime,
		"dedup_rate":           float64(s.logsDuplicate.Load()) / float64(s.logsReceived.Load()),
		"log_chan_size":        len(s.logChan),
		"log_chan_capacity":    cap(s.logChan),
	}
	
	json.NewEncoder(w).Encode(response)
}

func main() {
	clickhouseAddrEnv := os.Getenv("CLICKHOUSE_ADDR")
	if clickhouseAddrEnv != "" {
		clickhouseAddr = clickhouseAddrEnv
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// ClickHouse connection - dev mode (no authentication)
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseAddr},
		Auth: clickhouse.Auth{
			Database: database,
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

	encoder, _ := zstd.NewWriter(nil)
	decoder, _ := zstd.NewReader(nil)

	s := grpc.NewServer()
	server := &ingestionServer{
		db:         conn,
		logChan:    make(chan *pb.LogEntry, 1000),
		dedupCache: &sync.Map{},
		encoder:    encoder,
		decoder:    decoder,
		startTime:  time.Now(),
	}

	pb.RegisterLogIngestionServer(s, server)
	go server.batchWriter()

	// Start HTTP server for health and metrics
	http.HandleFunc("/health", server.healthHandler)
	http.HandleFunc("/metrics", server.metricsHandler)
	
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8082"
	}
	
	httpServer := &http.Server{
		Addr: ":" + httpPort,
	}
	
	go func() {
		log.Printf("Starting HTTP server on port %s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start gRPC server in a goroutine
	go func() {
		log.Printf("Ingestion server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	<-sigChan
	log.Println("Shutdown signal received, gracefully stopping...")
	
	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	
	// Gracefully stop gRPC server
	s.GracefulStop()
	
	// Close ClickHouse connection
	if conn != nil {
		conn.Close()
	}
	
	log.Println("Ingestion server stopped gracefully")
}
