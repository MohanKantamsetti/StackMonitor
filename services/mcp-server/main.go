package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type MCPServer struct {
	apiServerURL string
	geminiClient *genai.Client
	useLLM       bool
}

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Response string `json:"response"`
	Data     any    `json:"data,omitempty"`
}

func NewMCPServer(apiURL, geminiKey string, useLLM bool) (*MCPServer, error) {
	var client *genai.Client
	var err error

	if useLLM && geminiKey != "" {
		ctx := context.Background()
		client, err = genai.NewClient(ctx, option.WithAPIKey(geminiKey))
		if err != nil {
			log.Printf("Failed to create Gemini client: %v", err)
			useLLM = false
		}
	}

	return &MCPServer{
		apiServerURL: apiURL,
		geminiClient: client,
		useLLM:       useLLM,
	}, nil
}

func (m *MCPServer) handleMCPQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	log.Printf("Received query: %s", req.Query)

	var response string
	var data any

	if m.useLLM && m.geminiClient != nil {
		ctx := context.Background()
		model := m.geminiClient.GenerativeModel("gemini-1.5-flash")

		prompt := fmt.Sprintf(`You are an observability assistant. Analyze this query and determine the appropriate action:
Query: "%s"

Available tools:
1. get_logs - Fetch recent logs (parameters: limit, level, service)
2. get_error_rate - Get error rate metrics

Respond with JSON: {"tool": "tool_name", "params": {...}, "explanation": "..."}
If the query is conversational, respond with: {"tool": "none", "response": "your response"}`, req.Query)

		resp, err := model.GenerateContent(ctx, genai.Text(prompt))
		if err != nil {
			log.Printf("LLM error: %v", err)
			response, data = m.fallbackKeywordMatch(req.Query)
		} else {
			response, data = m.processLLMResponse(resp, req.Query)
		}
	} else {
		response, data = m.fallbackKeywordMatch(req.Query)
	}

	c.JSON(200, QueryResponse{
		Response: response,
		Data:     data,
	})
}

func (m *MCPServer) processLLMResponse(resp *genai.GenerateContentResponse, query string) (string, any) {
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return m.fallbackKeywordMatch(query)
	}

	text := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	text = strings.TrimSpace(text)
	
	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return m.fallbackKeywordMatch(query)
	}

	tool, ok := result["tool"].(string)
	if !ok || tool == "none" {
		if responseText, ok := result["response"].(string); ok {
			return responseText, nil
		}
		return m.fallbackKeywordMatch(query)
	}

	return m.executeTool(tool, result["params"])
}

func (m *MCPServer) fallbackKeywordMatch(query string) (string, any) {
	query = strings.ToLower(query)

	// Help/Fix/How queries - provide analysis and recommendations
	if strings.Contains(query, "fix") || strings.Contains(query, "how") || strings.Contains(query, "solve") || strings.Contains(query, "troubleshoot") {
		return m.analyzeAndRecommend()
	}

	// Error-related queries
	if strings.Contains(query, "error") && (strings.Contains(query, "rate") || strings.Contains(query, "trend")) {
		return m.executeTool("get_error_rate", nil)
	}

	// Log queries
	if strings.Contains(query, "error") || strings.Contains(query, "log") || strings.Contains(query, "recent") {
		params := map[string]any{"limit": "50"}
		if strings.Contains(query, "error") {
			params["level"] = "ERROR"
		}
		return m.executeTool("get_logs", params)
	}

	// Describe/summary queries
	if strings.Contains(query, "describe") || strings.Contains(query, "summary") || strings.Contains(query, "status") {
		params := map[string]any{"limit": "100"}
		return m.executeTool("get_logs", params)
	}

	return "I can help you with:\n• 🔍 Query logs: 'What are the recent errors?'\n• 📈 View trends: 'Show me error rate'\n• 💡 Get help: 'How to fix errors?'\n• 📊 System status: 'Describe my logs'", nil
}

func (m *MCPServer) executeTool(tool string, params any) (string, any) {
	switch tool {
	case "get_logs":
		limit := "100"
		level := ""
		service := ""

		if p, ok := params.(map[string]any); ok {
			if l, ok := p["limit"].(string); ok {
				limit = l
			}
			if lv, ok := p["level"].(string); ok {
				level = lv
			}
			if s, ok := p["service"].(string); ok {
				service = s
			}
		}

		url := fmt.Sprintf("%s/api/v1/logs?limit=%s", m.apiServerURL, limit)
		if level != "" {
			url += "&level=" + level
		}
		if service != "" {
			url += "&service=" + service
		}

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Sprintf("Error fetching logs: %v", err), nil
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var logs []map[string]any
		json.Unmarshal(body, &logs)

		// Generate descriptive response
		errorCount := 0
		warnCount := 0
		infoCount := 0
		for _, logEntry := range logs {
			if logLevel, ok := logEntry["level"].(string); ok {
				switch logLevel {
				case "ERROR":
					errorCount++
				case "WARN":
					warnCount++
				case "INFO":
					infoCount++
				}
			}
		}

		response := fmt.Sprintf("I found %d logs", len(logs))
		if level == "ERROR" {
			response = fmt.Sprintf("I found %d error logs", len(logs))
		} else if errorCount > 0 || warnCount > 0 {
			response = fmt.Sprintf("I found %d logs: %d errors, %d warnings, %d info", 
				len(logs), errorCount, warnCount, infoCount)
		}

		return response, logs

	case "get_error_rate":
		url := fmt.Sprintf("%s/api/v1/metrics/error-rate", m.apiServerURL)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Sprintf("Error fetching metrics: %v", err), nil
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var metrics []map[string]any
		json.Unmarshal(body, &metrics)

		// Calculate average error rate
		totalErrors := 0
		totalLogs := 0
		for _, metric := range metrics {
			if errors, ok := metric["errors"].(float64); ok {
				totalErrors += int(errors)
			}
			if total, ok := metric["total"].(float64); ok {
				totalLogs += int(total)
			}
		}

		errorRate := 0.0
		if totalLogs > 0 {
			errorRate = float64(totalErrors) / float64(totalLogs) * 100
		}

		response := fmt.Sprintf("Error rate analysis for the last hour: %.1f%% error rate (%d errors out of %d logs)", 
			errorRate, totalErrors, totalLogs)

		return response, metrics
	}

	return "Unknown tool", nil
}

func (m *MCPServer) analyzeAndRecommend() (string, any) {
	// Fetch recent error logs
	url := fmt.Sprintf("%s/api/v1/logs?limit=50&level=ERROR", m.apiServerURL)
	resp, err := http.Get(url)
	if err != nil {
		return "I couldn't fetch logs to analyze. Please check your connection.", nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var logs []map[string]any
	json.Unmarshal(body, &logs)

	if len(logs) == 0 {
		return "🎉 Great news! No errors found in recent logs. Your system is running smoothly!", nil
	}

	// Analyze error patterns
	errorTypes := make(map[string]int)
	errorExamples := make(map[string]string)
	
	for _, log := range logs {
		if msg, ok := log["message"].(string); ok {
			// Categorize errors by type
			errorType := categorizeError(msg)
			errorTypes[errorType]++
			if errorExamples[errorType] == "" {
				errorExamples[errorType] = msg
			}
		}
	}

	// Generate recommendations
	response := fmt.Sprintf("🔍 **Error Analysis** (last 50 errors):\n\n")
	
	// Find top error types
	type errorCount struct {
		Type  string
		Count int
	}
	var sortedErrors []errorCount
	for errType, count := range errorTypes {
		sortedErrors = append(sortedErrors, errorCount{errType, count})
	}
	
	// Sort by count (simple bubble sort for small dataset)
	for i := 0; i < len(sortedErrors); i++ {
		for j := i + 1; j < len(sortedErrors); j++ {
			if sortedErrors[j].Count > sortedErrors[i].Count {
				sortedErrors[i], sortedErrors[j] = sortedErrors[j], sortedErrors[i]
			}
		}
	}

	// Top 3 error types with recommendations
	maxTypes := 3
	if len(sortedErrors) < maxTypes {
		maxTypes = len(sortedErrors)
	}

	for i := 0; i < maxTypes; i++ {
		errType := sortedErrors[i].Type
		count := sortedErrors[i].Count
		percentage := float64(count) / float64(len(logs)) * 100
		
		response += fmt.Sprintf("**%d. %s** (%d errors, %.1f%%)\n", i+1, errType, count, percentage)
		response += fmt.Sprintf("   Example: `%s`\n", truncateString(errorExamples[errType], 80))
		response += fmt.Sprintf("   💡 **Fix**: %s\n\n", getRecommendation(errType))
	}

	if len(sortedErrors) > maxTypes {
		response += fmt.Sprintf("*...and %d more error types*\n\n", len(sortedErrors)-maxTypes)
	}

	response += "**🛠️ Quick Actions:**\n"
	response += "• Restart affected services\n"
	response += "• Check resource utilization (memory, disk)\n"
	response += "• Review recent deployments\n"
	response += "• Monitor error rates for trends"

	return response, logs
}

func categorizeError(message string) string {
	msg := strings.ToLower(message)
	
	if strings.Contains(msg, "connection") && (strings.Contains(msg, "refused") || strings.Contains(msg, "failed") || strings.Contains(msg, "timeout")) {
		return "🔌 Connection Issues"
	}
	if strings.Contains(msg, "memory") || strings.Contains(msg, "outofmemory") {
		return "💾 Memory Problems"
	}
	if strings.Contains(msg, "database") || strings.Contains(msg, "deadlock") || strings.Contains(msg, "query") {
		return "🗄️ Database Errors"
	}
	if strings.Contains(msg, "413") || strings.Contains(msg, "request entity too large") {
		return "📦 Request Size Limits"
	}
	if strings.Contains(msg, "500") || strings.Contains(msg, "502") || strings.Contains(msg, "503") {
		return "🌐 Server Errors"
	}
	if strings.Contains(msg, "payment") || strings.Contains(msg, "order") {
		return "💳 Payment/Order Issues"
	}
	if strings.Contains(msg, "s3") || strings.Contains(msg, "upload") || strings.Contains(msg, "accessdenied") {
		return "☁️ Cloud Storage Issues"
	}
	if strings.Contains(msg, "nullpointer") || strings.Contains(msg, "exception") {
		return "🐛 Application Bugs"
	}
	if strings.Contains(msg, "ssl") || strings.Contains(msg, "certificate") {
		return "🔐 SSL/Certificate Issues"
	}
	
	return "⚠️ General Errors"
}

func getRecommendation(errorType string) string {
	recommendations := map[string]string{
		"🔌 Connection Issues":    "Check network connectivity, verify service availability, increase timeout values",
		"💾 Memory Problems":       "Increase heap size, optimize memory usage, check for memory leaks",
		"🗄️ Database Errors":       "Review query performance, check connection pool settings, optimize indexes",
		"📦 Request Size Limits":   "Increase max request size in nginx/server config, implement chunked uploads",
		"🌐 Server Errors":         "Check upstream services, review logs, restart affected services",
		"💳 Payment/Order Issues":  "Verify payment gateway status, check API credentials, review transaction logs",
		"☁️ Cloud Storage Issues":  "Check IAM permissions, verify bucket policies, validate access keys",
		"🐛 Application Bugs":      "Review code at error line, add null checks, update dependencies",
		"🔐 SSL/Certificate Issues": "Renew certificates, update certificate chain, check certificate validity",
	}
	
	if rec, ok := recommendations[errorType]; ok {
		return rec
	}
	return "Review logs for patterns, check recent changes, monitor system resources"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func main() {
	apiServerURL := os.Getenv("API_SERVER_URL")
	if apiServerURL == "" {
		apiServerURL = "http://api-server:5000"
	}

	geminiKey := os.Getenv("GEMINI_API_KEY")
	useLLM := os.Getenv("USE_LLM") == "true"

	server, err := NewMCPServer(apiServerURL, geminiKey, useLLM)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.POST("/mcp/query", server.handleMCPQuery)

	log.Printf("MCP Server starting on :5001 (LLM enabled: %v)", useLLM)
	r.Run(":5001")
}

