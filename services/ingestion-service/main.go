package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	pb "stackmonitor.com/logproto"
	"google.golang.org/grpc"
)

type IngestionServer struct {
	pb.UnimplementedLogIngestionServer
	db          *sql.DB
	batchCache  map[int64]bool
	mu          sync.RWMutex
}

func NewIngestionServer(db *sql.DB) *IngestionServer {
	return &IngestionServer{
		db:         db,
		batchCache: make(map[int64]bool),
	}
}

func (s *IngestionServer) StreamLogs(stream pb.LogIngestion_StreamLogsServer) error {
	for {
		batch, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		
		// Check for duplicate
		s.mu.RLock()
		if s.batchCache[batch.BatchId] {
			s.mu.RUnlock()
			stream.Send(&pb.Ack{
				BatchId: batch.BatchId,
				Status:  pb.AckStatus_DROP,
				Message: "Duplicate batch",
			})
			continue
		}
		s.mu.RUnlock()
		
		// Insert logs
		if err := s.insertBatch(batch); err != nil {
			log.Printf("Error inserting batch: %v", err)
			stream.Send(&pb.Ack{
				BatchId: batch.BatchId,
				Status:  pb.AckStatus_RETRY,
				Message: err.Error(),
			})
			continue
		}
		
		// Mark as processed
		s.mu.Lock()
		s.batchCache[batch.BatchId] = true
		s.mu.Unlock()
		
		// Send ack
		stream.Send(&pb.Ack{
			BatchId:            batch.BatchId,
			Status:             pb.AckStatus_SUCCESS,
			Message:            "OK",
			ServerTimestampMs:  time.Now().UnixMilli(),
		})
	}
}

func (s *IngestionServer) insertBatch(batch *pb.LogBatch) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`
		INSERT INTO logs (timestamp, level, message, source, agent_id, fields)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	for _, entry := range batch.Logs {
		fieldsJSON := "{}"
		if len(entry.Fields) > 0 {
			// Simple JSON encoding
			fieldsJSON = "{"
			first := true
			for k, v := range entry.Fields {
				if !first {
					fieldsJSON += ","
				}
				fieldsJSON += fmt.Sprintf(`"%s":"%s"`, k, v)
				first = false
			}
			fieldsJSON += "}"
		}
		
		_, err = stmt.Exec(
			time.Unix(0, entry.TimestampNs),
			entry.Level,
			entry.Message,
			entry.Source,
			entry.AgentId,
			fieldsJSON,
		)
		if err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

func initClickHouse() (*sql.DB, error) {
	clickhouseURL := os.Getenv("CLICKHOUSE_URL")
	if clickhouseURL == "" {
		clickhouseURL = "clickhouse:9000"
	}
	
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{clickhouseURL},
	})
	
	// Create table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			timestamp DateTime64(9),
			level String,
			message String,
			source String,
			agent_id String,
			fields String
		) ENGINE = MergeTree()
		ORDER BY (timestamp, agent_id)
	`)
	if err != nil {
		return nil, err
	}
	
	return db, nil
}

func main() {
	// Wait for ClickHouse
	time.Sleep(5 * time.Second)
	
	db, err := initClickHouse()
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer db.Close()
	
	server := NewIngestionServer(db)
	
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	grpcServer := grpc.NewServer()
	pb.RegisterLogIngestionServer(grpcServer, server)
	
	log.Println("Ingestion server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

