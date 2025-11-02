package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	pb "stackmonitor.com/configproto"
	"google.golang.org/grpc"
)

type Config struct {
	Version      string                 `json:"version"`
	SampleRate   float64                `json:"sample_rate"`
	BufferSize   int                    `json:"buffer_size"`
	FlushInterval string                 `json:"flush_interval"`
	Tags         map[string]string      `json:"tags"`
}

type ConfigServer struct {
	pb.UnimplementedConfigServiceServer
	configs map[string]*Config
	mu      sync.RWMutex
	version string
}

func NewConfigServer() *ConfigServer {
	cs := &ConfigServer{
		configs: make(map[string]*Config),
		version: time.Now().Format("20060102150405"),
	}
	
	// Default configuration
	defaultConfig := &Config{
		Version:       cs.version,
		SampleRate:    0.1,
		BufferSize:    1000,
		FlushInterval: "30s",
		Tags: map[string]string{
			"environment": "production",
			"cluster":     "main",
		},
	}
	
	cs.configs["default"] = defaultConfig
	return cs
}

func (s *ConfigServer) GetConfig(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Get agent-specific config or default
	config, ok := s.configs[req.AgentId]
	if !ok {
		config = s.configs["default"]
	}
	
	// Check if version matches
	if req.CurrentConfigVersion == config.Version {
		return &pb.ConfigResponse{
			ConfigVersion: config.Version,
			ConfigPayload: nil, // No change
		}, nil
	}
	
	// Marshal config
	payload, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	
	return &pb.ConfigResponse{
		ConfigVersion: config.Version,
		ConfigPayload: payload,
	}, nil
}

func (s *ConfigServer) updateConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var config Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	s.mu.Lock()
	s.version = time.Now().Format("20060102150405")
	config.Version = s.version
	s.configs["default"] = &config
	s.mu.Unlock()
	
	log.Printf("Config updated to version %s", s.version)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"version": s.version})
}

func main() {
	configServer := NewConfigServer()
	
	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", ":50052")
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}
		
		grpcServer := grpc.NewServer()
		pb.RegisterConfigServiceServer(grpcServer, configServer)
		
		log.Println("Config gRPC server listening on :50052")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	
	// Start HTTP server for management
	http.HandleFunc("/update", configServer.updateConfigHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	log.Println("Config HTTP server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}

