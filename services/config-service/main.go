package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"

	pb "stackmonitor.com/config-service/proto/configproto"
)

const (
	port       = ":8080"
	configFile = "/config/config.yaml"
)

type configServer struct {
	pb.UnimplementedConfigServiceServer
	configPayload []byte
	configVersion string
	mu            sync.RWMutex
}

func (s *configServer) loadConfig() {
	payload, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("Failed to read config file: %v", err)
		return
	}

	hash := sha256.Sum256(payload)
	version := hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter version

	s.mu.Lock()
	oldVersion := s.configVersion
	s.configPayload = payload
	s.configVersion = version
	s.mu.Unlock()
	
	// Only log if version actually changed
	if oldVersion != "" && oldVersion != version {
		log.Printf("Loaded new config version: %s (previous: %s)", version, oldVersion)
	} else if oldVersion == "" {
		log.Printf("Loaded initial config version: %s", version)
	}
}

func (s *configServer) GetConfig(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if req.CurrentConfigVersion == s.configVersion {
		// Version is the same, send back empty payload
		return &pb.ConfigResponse{ConfigVersion: s.configVersion}, nil
	}

	// Send new config
	return &pb.ConfigResponse{
		ConfigVersion: s.configVersion,
		ConfigPayload: s.configPayload,
	}, nil
}

func main() {
	s := &configServer{}
	s.loadConfig()

	// Watch config file for changes (polling every 10s)
	go func() {
		for {
			time.Sleep(10 * time.Second) // Poll file every 10s
			s.loadConfig()                // Reload if changed
		}
	}()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterConfigServiceServer(grpcServer, s)

	log.Printf("Config server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
