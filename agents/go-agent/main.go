package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"

	configpb "stackmonitor.com/configproto"
	logpb "stackmonitor.com/logproto"
)

type AgentConfig struct {
	Version       string `yaml:"version"`
	AgentSettings struct {
		PollInterval string `yaml:"poll_interval"`
		BatchSizeKB  int    `yaml:"batch_size_kb"`
		BatchWindow  string `yaml:"batch_window"`
	} `yaml:"agent_settings"`
	Sampling struct {
		BaseRates map[string]float64 `yaml:"base_rates"`
		ContentRules []struct {
			Pattern string  `yaml:"pattern"`
			Rate    float64 `yaml:"rate"`
		} `yaml:"content_rules"`
	} `yaml:"sampling"`
}

type Agent struct {
	id              string
	configClient    configpb.ConfigServiceClient
	ingestionClient logpb.LogIngestionClient
	config          *AgentConfig
	configVersion   string
	mu              sync.RWMutex
	logChan         chan *logpb.LogEntry
	stream          logpb.LogIngestion_StreamLogsClient
	conn            *grpc.ClientConn
	batchID         int64
}

var appLogRegex = regexp.MustCompile(`^(\S+)\s+(\S+)\s+\[(\S+)\]\s+(.*)`)
var tomcatLogRegex = regexp.MustCompile(`^(\d{2}-[A-Za-z]{3}-\d{4}\s+\d{2}:\d{2}:\d{2}\.\d{3})\s+(\S+)\s+\[([^\]]+)\]\s+(.*)`)
var nginxLogRegex = regexp.MustCompile(`^(\S+)\s+-\s+-\s+\[([^\]]+)\]\s+"(\S+)\s+(\S+)\s+(\S+)"\s+(\d+)\s+(\d+)\s+"([^"]+)"\s+"([^"]+)"`)

func (a *Agent) parseLog(line, source string) *logpb.LogEntry {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	var t time.Time
	var level, service, message string
	var err error

	if matches := appLogRegex.FindStringSubmatch(line); matches != nil {
		t, err = time.Parse(time.RFC3339Nano, matches[1])
		if err != nil {
			t, err = time.Parse(time.RFC3339, matches[1])
		}
		if err == nil {
			level = matches[2]
			service = matches[3]
			message = matches[4]
		}
	} else if matches := tomcatLogRegex.FindStringSubmatch(line); matches != nil {
		t, err = time.Parse("02-Jan-2006 15:04:05.000", matches[1])
		if err == nil {
			levelStr := matches[2]
			switch levelStr {
			case "SEVERE":
				level = "ERROR"
			case "WARNING":
				level = "WARN"
			default:
				level = "INFO"
			}
			service = "tomcat"
			message = matches[4]
		}
	} else if matches := nginxLogRegex.FindStringSubmatch(line); matches != nil {
		t, err = time.Parse("02/Jan/2006:15:04:05 -0700", matches[2])
		if err == nil {
			statusCode := matches[6]
			statusInt := 0
			fmt.Sscanf(statusCode, "%d", &statusInt)
			if statusInt >= 500 {
				level = "ERROR"
			} else if statusInt >= 400 {
				level = "WARN"
			} else {
				level = "INFO"
			}
			service = "nginx"
			message = fmt.Sprintf("%s %s %s - Status: %s", matches[3], matches[4], matches[5], statusCode)
		}
	}

	if err != nil || t.IsZero() {
		return nil
	}

	a.mu.RLock()
	rate, ok := a.config.Sampling.BaseRates[level]
	if !ok {
		rate = 0.1
	}
	
	for _, rule := range a.config.Sampling.ContentRules {
		if strings.Contains(message, rule.Pattern) {
			rate = rule.Rate
			break
		}
	}
	a.mu.RUnlock()

	if rate < 1.0 {
		n, _ := rand.Int(rand.Reader, big.NewInt(100))
		if n.Int64() > int64(rate*100) {
			return nil
		}
	}

	return &logpb.LogEntry{
		TimestampNs: t.UnixNano(),
		Level:       level,
		Message:     message,
		Source:      source,
		Fields: map[string]string{
			"service":  service,
			"trace_id": fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		},
		AgentId: a.id,
	}
}

func (a *Agent) tailFile(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Failed to create watcher for %s: %v", path, err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		log.Printf("Failed to watch %s: %v", path, err)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open %s: %v", path, err)
		return
	}
	defer file.Close()

	file.Seek(0, io.SeekEnd)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				data := make([]byte, 4096)
				n, err := file.Read(data)
				if err != nil && err != io.EOF {
					log.Printf("Error reading file: %v", err)
					continue
				}
				if n > 0 {
					lines := strings.Split(string(data[:n]), "\n")
					for _, line := range lines {
						if line != "" {
							entry := a.parseLog(line, path)
							if entry != nil {
								a.logChan <- entry
							}
						}
					}
				}
			}
		case err := <-watcher.Errors:
			log.Printf("Watcher error for %s: %v", path, err)
		}
	}
}

func (a *Agent) batchSender() {
	ctx := context.Background()
	stream, err := a.ingestionClient.StreamLogs(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}
	a.stream = stream

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	buffer := make([]*logpb.LogEntry, 0, 100)

	go func() {
		for {
			ack, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("Error receiving ack: %v", err)
				return
			}
			log.Printf("Received ack for batch %d: %s", ack.BatchId, ack.Message)
		}
	}()

	for {
		select {
		case entry := <-a.logChan:
			buffer = append(buffer, entry)
			if len(buffer) >= 100 {
				a.sendBatch(buffer)
				buffer = make([]*logpb.LogEntry, 0, 100)
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				a.sendBatch(buffer)
				buffer = make([]*logpb.LogEntry, 0, 100)
			}
		}
	}
}

func (a *Agent) sendBatch(logs []*logpb.LogEntry) {
	if len(logs) == 0 {
		return
	}

	a.batchID++
	batch := &logpb.LogBatch{
		AgentId:      a.id,
		BatchId:      a.batchID,
		TimestampMs:  time.Now().UnixMilli(),
		Logs:         logs,
		Compression:  logpb.CompressionType_NONE,
		Metadata:     make(map[string]string),
	}

	if err := a.stream.Send(batch); err != nil {
		log.Printf("Failed to send batch: %v", err)
	} else {
		log.Printf("Sent batch %d with %d logs", a.batchID, len(logs))
	}
}

func (a *Agent) configPoller() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		a.mu.RLock()
		currentVersion := a.configVersion
		a.mu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := a.configClient.GetConfig(ctx, &configpb.ConfigRequest{
			AgentId:             a.id,
			CurrentConfigVersion: currentVersion,
		})
		cancel()

		if err != nil {
			log.Printf("Failed to get config: %v", err)
		} else if resp.ConfigVersion != currentVersion && len(resp.ConfigPayload) > 0 {
			var newConfig AgentConfig
			if err := yaml.Unmarshal(resp.ConfigPayload, &newConfig); err == nil {
				a.mu.Lock()
				a.config = &newConfig
				a.configVersion = resp.ConfigVersion
				a.mu.Unlock()
				log.Printf("Config reloaded to version %s", newConfig.Version)
			}
		}

		<-ticker.C
	}
}

func main() {
	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		agentID = fmt.Sprintf("go-agent-%d", time.Now().Unix())
	}

	configURL := os.Getenv("CONFIG_URL")
	if configURL == "" {
		configURL = "config-service:8080"
	}

	ingestionURL := os.Getenv("INGESTION_URL")
	if ingestionURL == "" {
		ingestionURL = "ingestion-service:50051"
	}

	configConn, err := grpc.Dial(configURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to config service: %v", err)
	}
	defer configConn.Close()
	configClient := configpb.NewConfigServiceClient(configConn)

	ingestionConn, err := grpc.Dial(ingestionURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to ingestion service: %v", err)
	}
	defer ingestionConn.Close()
	ingestionClient := logpb.NewLogIngestionClient(ingestionConn)

	agent := &Agent{
		id:              agentID,
		configClient:    configClient,
		ingestionClient: ingestionClient,
		conn:            ingestionConn,
		logChan:         make(chan *logpb.LogEntry, 1000),
		config:          &AgentConfig{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := configClient.GetConfig(ctx, &configpb.ConfigRequest{
		AgentId:             agentID,
		CurrentConfigVersion: "",
	})
	cancel()

	if err == nil && len(resp.ConfigPayload) > 0 {
		var cfg AgentConfig
		if err := yaml.Unmarshal(resp.ConfigPayload, &cfg); err == nil {
			agent.config = &cfg
			agent.configVersion = resp.ConfigVersion
			log.Printf("Loaded initial config version: %s", resp.ConfigVersion)
		}
	}

	go agent.configPoller()
	go agent.batchSender()

	logFiles := []string{"/logs/app.log", "/logs/tomcat.log", "/logs/nginx.log"}
	for _, file := range logFiles {
		if _, err := os.Stat(file); err == nil {
			go agent.tailFile(file)
			log.Printf("Started tailing %s", file)
		} else {
			log.Printf("Log file %s not found, skipping", file)
		}
	}

	log.Println("Go agent started. Waiting for logs...")
	select {}
}
