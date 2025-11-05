package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/klauspost/compress/zstd"
	"os"
	"google.golang.org/grpc"

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
}

// PoC Deduplication: simple in-memory hash cache
func (s *ingestionServer) isDuplicate(entry *pb.LogEntry) bool {
	// Include agent_id and trace_id in hash for better duplicate detection
	hash := fmt.Sprintf("%d-%s-%s-%s-%s", entry.TimestampNs, entry.Message, entry.Level, entry.AgentId, entry.Fields["trace_id"])
	if _, loaded := s.dedupCache.LoadOrStore(hash, true); loaded {
		return true
	}
	// Expire old cache entries (simple time-based) - extended to 60s for testing
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

		// Use logs directly from batch
		logsToProcess := batch.Logs
		
		// Handle compression if needed (for future use)
		if batch.Compression == pb.CompressionType_ZSTD && len(batch.CompressedPayload) > 0 {
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
			// For PoC, we use uncompressed logs
			_ = decompressed // Suppress unused
		}

		processedCount := 0
		duplicateCount := 0
		for _, entry := range logsToProcess {
			// Apply deduplication
			if !s.isDuplicate(entry) {
				s.logChan <- entry
				processedCount++
			} else {
				duplicateCount++
			}
		}
		log.Printf("📥 Received batch %d: %d logs (processed: %d, duplicates: %d)", batch.BatchId, len(logsToProcess), processedCount, duplicateCount)

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
		return
	}
	log.Printf("✅ Inserted %d logs into ClickHouse", len(logs))
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
	}

	pb.RegisterLogIngestionServer(s, server)
	go server.batchWriter()

	log.Printf("Ingestion server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
