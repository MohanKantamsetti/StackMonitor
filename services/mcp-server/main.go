package main

import (
	"bytes"
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

const apiServerURL = "http://api-server:5000/api/v1"

type MCPServer struct {
	geminiClient *genai.Client
	apiServerURL string
	useLLM       bool
}

func NewMCPServer() *MCPServer {
	apiKey := os.Getenv("GEMINI_API_KEY")
	useLLM := apiKey != "" && os.Getenv("USE_LLM") == "true"

	var client *genai.Client
	if useLLM {
		ctx := context.Background()
		var err error
		client, err = genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			log.Printf("Failed to initialize Gemini client: %v", err)
			useLLM = false
		} else {
			log.Println("MCP Server initialized with Google Gemini LLM")
		}
	} else {
		log.Println("MCP Server initialized with keyword matching (set GEMINI_API_KEY and USE_LLM=true for LLM)")
	}

	return &MCPServer{
		geminiClient: client,
		apiServerURL: apiServerURL,
		useLLM:       useLLM,
	}
}

// PoC simulation of MCP tool calling with optional LLM
func (mcp *MCPServer) handleMCPQuery(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	query := req.Query
	var response string

	// Check if query is asking for analysis/summary (should use LLM with data)
	queryLower := strings.ToLower(query)
	needsAnalysis := strings.Contains(queryLower, "what are") || strings.Contains(queryLower, "what is") ||
		strings.Contains(queryLower, "summarize") || strings.Contains(queryLower, "summary") ||
		strings.Contains(queryLower, "analyze") || strings.Contains(queryLower, "analysis") ||
		strings.Contains(queryLower, "most") || strings.Contains(queryLower, "common") ||
		strings.Contains(queryLower, "tell me about") || strings.Contains(queryLower, "explain")
	
	if needsAnalysis {
		// For analysis queries, fetch data first, then pass to LLM
		response = mcp.processAnalysisQuery(query)
	} else {
		// First try keyword matching for known patterns
		keywordResponse, hasKeywordMatch := mcp.tryKeywordMatching(query)
		if hasKeywordMatch {
			response = keywordResponse
		} else {
			// No keyword match - always try LLM if API key is available
			// This allows natural language queries to be handled by AI
			response = mcp.processWithGemini(query)
			
			// If LLM failed and we have a keyword fallback, use it
			if strings.Contains(response, "Error") || strings.Contains(response, "trouble connecting") {
				if keywordResponse != "" {
					response = keywordResponse + "\n\n" + response // Combine both responses
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func (mcp *MCPServer) processWithGemini(query string) string {
	// Always try to initialize if API key is available (even if USE_LLM wasn't set)
	if mcp.geminiClient == nil {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey != "" {
			ctx := context.Background()
			client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
			if err != nil {
				log.Printf("Failed to initialize Gemini client: %v", err)
				return "I'm not sure how to answer that. Try asking about:\n• 'show me errors' or 'what errors do we have?'\n• 'show warnings'\n• 'what are the recent logs?'\n• 'show metrics' or 'error rate'\n• 'how can I fix these errors?'"
			}
			mcp.geminiClient = client
			log.Println("Gemini client initialized for query")
		} else {
			return "I'm not sure how to answer that. Try asking about:\n• 'show me errors' or 'what errors do we have?'\n• 'show warnings'\n• 'what are the recent logs?'\n• 'show metrics' or 'error rate'\n• 'how can I fix these errors?'"
		}
	}

	ctx := context.Background()

	// Create a system prompt that explains the available tools and provides context
	systemPrompt := `You are an observability assistant for StackMonitor, a log monitoring and analysis platform. You help users understand their system health through logs and metrics.

You have access to a log monitoring system with:
- Error, warning, and info logs from various services
- Metrics and performance data
- System statistics and health information

Provide helpful, natural language responses to user questions. You can:
- Answer questions about system health, errors, warnings, and performance
- Provide recommendations for fixing issues
- Explain what different error types mean
- Help users understand their system's behavior
- Have general conversations about observability and monitoring

Be conversational, helpful, and technical when appropriate. If the user asks something unrelated to logs/monitoring, you can still provide a helpful response.`

	// Combine system prompt and user query
	fullPrompt := systemPrompt + "\n\nUser query: " + query
	
	// First, try to list available models to find a working one
	var workingModelName string
	iter := mcp.geminiClient.ListModels(ctx)
	for {
		model, err := iter.Next()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Error listing models: %v", err)
			break
		}
		// Check if model supports generateContent
		if model != nil && model.SupportedGenerationMethods != nil {
			for _, method := range model.SupportedGenerationMethods {
				if method == "generateContent" {
					workingModelName = model.Name
					// Remove "models/" prefix if present
					if strings.HasPrefix(workingModelName, "models/") {
						workingModelName = strings.TrimPrefix(workingModelName, "models/")
					}
					log.Printf("Found working model: %s", workingModelName)
					break
				}
			}
			if workingModelName != "" {
				break
			}
		}
	}
	
	// If we couldn't list models, try common model names
	if workingModelName == "" {
		modelNames := []string{"gemini-1.5-flash", "gemini-1.5-pro", "gemini-pro"}
		for _, name := range modelNames {
			workingModelName = name
			log.Printf("Trying model: %s", workingModelName)
			break // Try the first one
		}
	}
	
	// Generate content with the working model
	var resp *genai.GenerateContentResponse
	var err error
	
	if workingModelName != "" {
		model := mcp.geminiClient.GenerativeModel(workingModelName)
		resp, err = model.GenerateContent(ctx, genai.Text(fullPrompt))
		if err != nil {
			log.Printf("Error with model %s: %v", workingModelName, err)
			// Try fallback models
			fallbackModels := []string{"gemini-1.5-flash", "gemini-1.5-pro"}
			for _, fallbackName := range fallbackModels {
				if fallbackName == workingModelName {
					continue // Skip the one we already tried
				}
				log.Printf("Trying fallback model: %s", fallbackName)
				model = mcp.geminiClient.GenerativeModel(fallbackName)
				resp, err = model.GenerateContent(ctx, genai.Text(fullPrompt))
				if err == nil {
					workingModelName = fallbackName
					break // Success!
				}
				log.Printf("Fallback model %s also failed: %v", fallbackName, err)
			}
		}
	}
	
	if err != nil || resp == nil {
		log.Printf("All Gemini models failed, last error: %v", err)
		return fmt.Sprintf("I'm having trouble connecting to the AI service. Here are some things you can ask:\n\n• 'show me errors' or 'what errors do we have?'\n• 'show warnings'\n• 'what are the recent logs?'\n• 'show metrics' or 'error rate'\n• 'how can I fix these errors?'\n\nError: %v", err)
	}

	// Extract response text
	var llmResponse strings.Builder
	if resp != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if text, ok := part.(genai.Text); ok {
					llmResponse.WriteString(string(text))
				}
			}
		}
	}

	responseText := llmResponse.String()
	if responseText == "" {
		log.Printf("Empty response from Gemini")
		return "I received an empty response from the AI service. Please try rephrasing your question or ask about 'errors', 'warnings', or 'metrics'."
	}
	
	log.Printf("Gemini response: %s", responseText)

	// For general queries, return the LLM response directly
	// Only extract tool calls if the query seems to want specific data
	queryLower := strings.ToLower(query)
	needsData := strings.Contains(queryLower, "show") || strings.Contains(queryLower, "get") || 
		strings.Contains(queryLower, "list") || strings.Contains(queryLower, "find") ||
		strings.Contains(queryLower, "what are") || strings.Contains(queryLower, "what is")

	if needsData {
		// Try to extract tool call intent from Gemini response
		toolCallURL, _ := mcp.extractToolFromLLMResponse(responseText, query)
		if toolCallURL != "" {
			// Call the tool and append results
			toolResult, err := mcp.callTool(toolCallURL)
			if err == nil {
				return fmt.Sprintf("%s\n\n**Data:**\n%s", responseText, toolResult)
			}
		}
	}

	// Return the LLM response directly
	return responseText
}

// Process analysis queries - fetch data and analyze with LLM
func (mcp *MCPServer) processAnalysisQuery(query string) string {
	queryLower := strings.ToLower(query)
	
	// Determine what data to fetch based on query
	var dataType string
	var toolURL string
	var dataJSON string
	
	if strings.Contains(queryLower, "error") {
		dataType = "errors"
		toolURL = fmt.Sprintf("%s/logs?level=ERROR&limit=50", mcp.apiServerURL)
	} else if strings.Contains(queryLower, "warn") {
		dataType = "warnings"
		toolURL = fmt.Sprintf("%s/logs?level=WARN&limit=50", mcp.apiServerURL)
	} else {
		// Default to errors if unclear
		dataType = "errors"
		toolURL = fmt.Sprintf("%s/logs?level=ERROR&limit=50", mcp.apiServerURL)
	}
	
	// Fetch the data
	dataResult, err := mcp.callTool(toolURL)
	if err != nil {
		return fmt.Sprintf("❌ Error fetching %s: %v", dataType, err)
	}
	dataJSON = dataResult
	
	// Parse to check if we have data
	var data struct {
		Logs []struct {
			Level   string `json:"level"`
			Service string `json:"service"`
			Message string `json:"message"`
			Timestamp string `json:"timestamp"`
		} `json:"logs"`
		Count int `json:"count"`
	}
	
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return fmt.Sprintf("❌ Error parsing data: %v", err)
	}
	
	if len(data.Logs) == 0 {
		return fmt.Sprintf("✅ No %s found. Your system looks healthy!", dataType)
	}
	
	// Initialize LLM client if needed
	if mcp.geminiClient == nil {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey != "" {
			ctx := context.Background()
			client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
			if err != nil {
				log.Printf("Failed to initialize Gemini client: %v", err)
				// Fallback to keyword-based analysis
				return mcp.analyzeErrorsAndRecommend(dataJSON)
			}
			mcp.geminiClient = client
		} else {
			// Fallback to keyword-based analysis
			return mcp.analyzeErrorsAndRecommend(dataJSON)
		}
	}
	
	// Prepare prompt with data
	analysisPrompt := fmt.Sprintf(`You are analyzing log data from a system monitoring platform. 

The user asked: "%s"

Here are the %s (total: %d):

%s

Please provide a comprehensive analysis that answers:
1. What are the most common types of errors/issues?
2. What patterns do you see?
3. What are the main causes?
4. What services are most affected?
5. Any recommendations?

Format your response in a clear, structured way with headings and bullet points. Be specific and actionable.`, 
		query, dataType, data.Count, mcp.formatLogsForAnalysis(data.Logs))
	
	// Get LLM response
	ctx := context.Background()
	var workingModelName string
	iter := mcp.geminiClient.ListModels(ctx)
	for {
		model, err := iter.Next()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			break
		}
		if model != nil && model.SupportedGenerationMethods != nil {
			for _, method := range model.SupportedGenerationMethods {
				if method == "generateContent" {
					workingModelName = model.Name
					if strings.HasPrefix(workingModelName, "models/") {
						workingModelName = strings.TrimPrefix(workingModelName, "models/")
					}
					break
				}
			}
			if workingModelName != "" {
				break
			}
		}
	}
	
	if workingModelName == "" {
		workingModelName = "gemini-1.5-flash"
	}
	
	model := mcp.geminiClient.GenerativeModel(workingModelName)
	resp, err := model.GenerateContent(ctx, genai.Text(analysisPrompt))
	if err != nil {
		// Try fallback
		model = mcp.geminiClient.GenerativeModel("gemini-1.5-pro")
		resp, err = model.GenerateContent(ctx, genai.Text(analysisPrompt))
		if err != nil {
			log.Printf("LLM analysis failed: %v, using fallback", err)
			return mcp.analyzeErrorsAndRecommend(dataJSON)
		}
	}
	
	// Extract LLM response
	var llmResponse strings.Builder
	if resp != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if text, ok := part.(genai.Text); ok {
					llmResponse.WriteString(string(text))
				}
			}
		}
	}
	
	responseText := llmResponse.String()
	if responseText == "" {
		return mcp.analyzeErrorsAndRecommend(dataJSON)
	}
	
	return responseText
}

// Format logs for analysis prompt
func (mcp *MCPServer) formatLogsForAnalysis(logs []struct {
	Level     string `json:"level"`
	Service   string `json:"service"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Total logs: %d\n\n", len(logs)))
	
	for i, log := range logs {
		if i >= 50 { // Limit to 50 for prompt
			result.WriteString(fmt.Sprintf("\n... and %d more logs", len(logs)-50))
			break
		}
		result.WriteString(fmt.Sprintf("- [%s] %s: %s\n", log.Level, log.Service, log.Message))
	}
	
	return result.String()
}

func (mcp *MCPServer) extractToolFromLLMResponse(llmResponse, originalQuery string) (string, string) {
	// Simple extraction: look for keywords in LLM response + original query
	lowerResponse := strings.ToLower(llmResponse + " " + originalQuery)

	if strings.Contains(lowerResponse, "error") && !strings.Contains(lowerResponse, "rate") {
		service := "payment-service"
		if strings.Contains(lowerResponse, "user") {
			service = "user-service"
		}
		return fmt.Sprintf("%s/logs?service=%s&level=ERROR&limit=10", mcp.apiServerURL, service), "logs"
	}

	if strings.Contains(lowerResponse, "metric") || strings.Contains(lowerResponse, "rate") {
		service := "payment-service"
		if strings.Contains(lowerResponse, "user") {
			service = "user-service"
		}
		return fmt.Sprintf("%s/metrics/error-rate?service=%s&range=1h", mcp.apiServerURL, service), "metrics"
	}

	if strings.Contains(lowerResponse, "log") && strings.Contains(lowerResponse, "recent") {
		return fmt.Sprintf("%s/logs?limit=20", mcp.apiServerURL), "logs"
	}

	return "", ""
}

func (mcp *MCPServer) callTool(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Pretty print JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return string(body), nil
	}

	return prettyJSON.String(), nil
}

// Try keyword matching first, returns response and whether it matched
func (mcp *MCPServer) tryKeywordMatching(query string) (string, bool) {
	queryLower := strings.ToLower(query)
	
	// Check for "fix" or "how to fix" queries
	hasFixKeywords := strings.Contains(queryLower, "fix") || strings.Contains(queryLower, "how to") ||
		strings.Contains(queryLower, "solution") || strings.Contains(queryLower, "resolve") ||
		strings.Contains(queryLower, "recommend") || strings.Contains(queryLower, "advice")

	// Enhanced Intent Recognition - handle various ways of asking about errors/issues
	hasErrorKeywords := strings.Contains(queryLower, "error") || strings.Contains(queryLower, "errors") ||
		strings.Contains(queryLower, "issue") || strings.Contains(queryLower, "issues") ||
		strings.Contains(queryLower, "problem") || strings.Contains(queryLower, "problems") ||
		strings.Contains(queryLower, "sus") || strings.Contains(queryLower, "suspicious") ||
		strings.Contains(queryLower, "fail") || strings.Contains(queryLower, "failing") ||
		strings.Contains(queryLower, "broken") || strings.Contains(queryLower, "break")

	hasWarningKeywords := strings.Contains(queryLower, "warn") || strings.Contains(queryLower, "warning") ||
		strings.Contains(queryLower, "warning")

	hasMetricKeywords := strings.Contains(queryLower, "metric") || strings.Contains(queryLower, "rate") ||
		strings.Contains(queryLower, "stat") || strings.Contains(queryLower, "stats") ||
		strings.Contains(queryLower, "performance") || strings.Contains(queryLower, "throughput")

	hasLogKeywords := strings.Contains(queryLower, "log") || strings.Contains(queryLower, "recent") ||
		strings.Contains(queryLower, "latest") || 
		(strings.Contains(queryLower, "what") && (strings.Contains(queryLower, "log") || strings.Contains(queryLower, "error") || strings.Contains(queryLower, "warning")))

	// If no keywords match, return false
	// Only match if it's clearly about logs/errors/warnings/metrics
	if !hasFixKeywords && !hasErrorKeywords && !hasWarningKeywords && !hasMetricKeywords && !hasLogKeywords {
		return "", false
	}

	// Process with keywords
	response := mcp.processWithKeywords(query)
	return response, true
}

func (mcp *MCPServer) processWithKeywords(query string) string {
	queryLower := strings.ToLower(query)
	var toolCallURL string
	var response string

	log.Printf("Received query: %s", query)

	// Check for "fix" or "how to fix" queries
	hasFixKeywords := strings.Contains(queryLower, "fix") || strings.Contains(queryLower, "how to") ||
		strings.Contains(queryLower, "solution") || strings.Contains(queryLower, "resolve") ||
		strings.Contains(queryLower, "recommend") || strings.Contains(queryLower, "advice")

	// Enhanced Intent Recognition - handle various ways of asking about errors/issues
	hasErrorKeywords := strings.Contains(queryLower, "error") || strings.Contains(queryLower, "errors") ||
		strings.Contains(queryLower, "issue") || strings.Contains(queryLower, "issues") ||
		strings.Contains(queryLower, "problem") || strings.Contains(queryLower, "problems") ||
		strings.Contains(queryLower, "sus") || strings.Contains(queryLower, "suspicious") ||
		strings.Contains(queryLower, "fail") || strings.Contains(queryLower, "failing") ||
		strings.Contains(queryLower, "broken") || strings.Contains(queryLower, "break")

	hasWarningKeywords := strings.Contains(queryLower, "warn") || strings.Contains(queryLower, "warning") ||
		strings.Contains(queryLower, "warning")

	hasMetricKeywords := strings.Contains(queryLower, "metric") || strings.Contains(queryLower, "rate") ||
		strings.Contains(queryLower, "stat") || strings.Contains(queryLower, "stats") ||
		strings.Contains(queryLower, "performance") || strings.Contains(queryLower, "throughput")

	hasLogKeywords := strings.Contains(queryLower, "log") || strings.Contains(queryLower, "recent") ||
		strings.Contains(queryLower, "latest") || strings.Contains(queryLower, "what")

	// Check for service-specific queries
	service := ""
	if strings.Contains(queryLower, "user service") || strings.Contains(queryLower, "user-service") {
		service = "user-service"
	} else if strings.Contains(queryLower, "payment service") || strings.Contains(queryLower, "payment-service") {
		service = "payment-service"
	} else if strings.Contains(queryLower, "tomcat") {
		service = "tomcat"
	} else if strings.Contains(queryLower, "nginx") {
		service = "nginx"
	}

	// Build query URL based on intent
	if hasFixKeywords && hasErrorKeywords {
		// User wants to know how to fix errors - analyze and provide recommendations
		toolCallURL := fmt.Sprintf("%s/logs?level=ERROR&limit=50", mcp.apiServerURL)
		toolResult, err := mcp.callTool(toolCallURL)
		if err != nil {
			response = fmt.Sprintf("❌ Error querying logs: %v", err)
		} else {
			recommendations := mcp.analyzeErrorsAndRecommend(toolResult)
			response = fmt.Sprintf("🔧 **Error Analysis & Recommendations:**\n\n%s", recommendations)
		}
	} else if hasFixKeywords {
		// User wants to fix something but didn't specify - get all errors and warnings
		errorURL := fmt.Sprintf("%s/logs?level=ERROR&limit=30", mcp.apiServerURL)
		warnURL := fmt.Sprintf("%s/logs?level=WARN&limit=30", mcp.apiServerURL)
		
		errorResult, err1 := mcp.callTool(errorURL)
		warnResult, err2 := mcp.callTool(warnURL)
		
		if err1 != nil && err2 != nil {
			response = fmt.Sprintf("❌ Error querying logs: %v", err1)
		} else {
			allIssues := ""
			if err1 == nil {
				formatted := mcp.formatLogResponse(errorResult, "errors")
				if formatted != "" {
					allIssues += "🔴 **Errors:**\n" + formatted + "\n\n"
				}
			}
			if err2 == nil {
				formatted := mcp.formatLogResponse(warnResult, "warnings")
				if formatted != "" {
					allIssues += "⚠️ **Warnings:**\n" + formatted + "\n\n"
				}
			}
			
			if allIssues == "" {
				response = "✅ No errors or warnings found. Your system is healthy!"
			} else {
				recommendations := mcp.analyzeErrorsAndRecommend(errorResult)
				response = fmt.Sprintf("%s🔧 **Recommendations:**\n\n%s", allIssues, recommendations)
			}
		}
	} else if hasErrorKeywords {
		// Query errors
		if service != "" {
			toolCallURL = fmt.Sprintf("%s/logs?service=%s&level=ERROR&limit=20", mcp.apiServerURL, service)
		} else {
			toolCallURL = fmt.Sprintf("%s/logs?level=ERROR&limit=20", mcp.apiServerURL)
		}

		toolResult, err := mcp.callTool(toolCallURL)
		if err != nil {
			response = fmt.Sprintf("❌ Error querying logs: %v", err)
		} else {
			// Parse the response to get count
			var data struct {
				Count int `json:"count"`
			}
			json.Unmarshal([]byte(toolResult), &data)
			
			// Format with API link
			formatted := mcp.formatLogResponse(toolResult, "errors")
			if formatted == "" {
				response = "✅ No errors found in recent logs. Your system looks healthy!"
			} else {
				response = fmt.Sprintf("🔴 **Recent Errors Found**\n\n%s", formatted)
			}
		}

	} else if hasWarningKeywords {
		// Query warnings
		if service != "" {
			toolCallURL = fmt.Sprintf("%s/logs?service=%s&level=WARN&limit=20", mcp.apiServerURL, service)
		} else {
			toolCallURL = fmt.Sprintf("%s/logs?level=WARN&limit=20", mcp.apiServerURL)
		}

		toolResult, err := mcp.callTool(toolCallURL)
		if err != nil {
			response = fmt.Sprintf("❌ Error querying logs: %v", err)
		} else {
			formatted := mcp.formatLogResponse(toolResult, "warnings")
			if formatted == "" {
				response = "✅ No warnings found in recent logs."
			} else {
				response = fmt.Sprintf("⚠️ **Found Warnings:**\n\n%s", formatted)
			}
		}

	} else if hasMetricKeywords {
		// Query metrics
		if service != "" {
			toolCallURL = fmt.Sprintf("%s/metrics/error-rate?service=%s&range=1h", mcp.apiServerURL, service)
		} else {
			toolCallURL = fmt.Sprintf("%s/metrics/error-rate?range=1h", mcp.apiServerURL)
		}

		toolResult, err := mcp.callTool(toolCallURL)
		if err != nil {
			response = fmt.Sprintf("❌ Error querying metrics: %v", err)
		} else {
			response = fmt.Sprintf("📊 **Error Rate Metrics:**\n\n%s", toolResult)
		}

	} else if hasLogKeywords || queryLower == "" {
		// Query recent logs (default)
		if service != "" {
			toolCallURL = fmt.Sprintf("%s/logs?service=%s&limit=20", mcp.apiServerURL, service)
		} else {
			toolCallURL = fmt.Sprintf("%s/logs?limit=20", mcp.apiServerURL)
		}

		toolResult, err := mcp.callTool(toolCallURL)
		if err != nil {
			response = fmt.Sprintf("❌ Error querying logs: %v", err)
		} else {
			formatted := mcp.formatLogResponse(toolResult, "logs")
			if formatted == "" {
				response = "📋 No recent logs found."
			} else {
				response = fmt.Sprintf("📋 **Recent Logs:**\n\n%s", formatted)
			}
		}

	} else {
		// Try to get stats as a fallback
		toolCallURL = fmt.Sprintf("%s/logs/stats", mcp.apiServerURL)
		toolResult, err := mcp.callTool(toolCallURL)
		if err != nil {
			response = fmt.Sprintf("I'm not sure how to answer that. Try asking about:\n- 'errors' or 'issues'\n- 'warnings'\n- 'metrics' or 'stats'\n- 'recent logs'\n\nError: %v", err)
		} else {
			response = fmt.Sprintf("📊 **System Status:**\n\n%s\n\nTry asking about 'errors', 'warnings', or 'recent logs' for more details.", toolResult)
		}
	}

	return response
}

// Format log response to be more readable with API links
func (mcp *MCPServer) formatLogResponse(jsonResponse, logType string) string {
	var data struct {
		Logs []struct {
			Timestamp string `json:"timestamp"`
			Level     string `json:"level"`
			Service   string `json:"service"`
			Message   string `json:"message"`
			TraceID   string `json:"trace_id"`
		} `json:"logs"`
		Count int `json:"count"`
	}

	if err := json.Unmarshal([]byte(jsonResponse), &data); err != nil {
		// If parsing fails, return the raw JSON
		return jsonResponse
	}

	if len(data.Logs) == 0 {
		return ""
	}

	var result strings.Builder
	
	// Calculate service breakdown
	serviceCount := make(map[string]int)
	for _, log := range data.Logs {
		serviceCount[log.Service]++
	}
	
	// Summary first
	result.WriteString(fmt.Sprintf("## 📊 Summary\n\n"))
	result.WriteString(fmt.Sprintf("**Total %s:** %d\n\n", logType, data.Count))
	
	if len(serviceCount) > 0 {
		result.WriteString("**By Service:**\n")
		for service, count := range serviceCount {
			result.WriteString(fmt.Sprintf("- %s: %d\n", service, count))
		}
		result.WriteString("\n")
	}

	// Show only first 3 logs inline for preview
	displayCount := 3
	if len(data.Logs) < displayCount {
		displayCount = len(data.Logs)
	}

	result.WriteString("## 🔍 Recent Examples\n\n")
	for i := 0; i < displayCount; i++ {
		log := data.Logs[i]
		// Truncate message if too long
		message := log.Message
		if len(message) > 120 {
			message = message[:120] + "..."
		}
		result.WriteString(fmt.Sprintf("%d. `[%s]` **%s**: %s\n", i+1, log.Level, log.Service, message))
	}

	// Add API link to view all
	if data.Count > displayCount {
		result.WriteString(fmt.Sprintf("\n_... and **%d more %s**_\n\n", data.Count-displayCount, logType))
	}
	
	// Generate API query link based on log type
	apiURL := fmt.Sprintf("http://localhost:5000/api/v1/logs?limit=%d", data.Count)
	if logType == "errors" {
		apiURL = fmt.Sprintf("http://localhost:5000/api/v1/logs?level=ERROR&limit=%d", data.Count)
	} else if logType == "warnings" {
		apiURL = fmt.Sprintf("http://localhost:5000/api/v1/logs?level=WARN&limit=%d", data.Count)
	}
	
	result.WriteString("\n---\n\n")
	result.WriteString(fmt.Sprintf("### 🔗 View Full Details\n\n"))
	result.WriteString(fmt.Sprintf("**[📊 Open all %d %s in API (New Tab) →](%s)**\n\n", data.Count, logType, apiURL))
	result.WriteString(fmt.Sprintf("This link opens the complete API response with all logs, timestamps, and trace IDs.\n"))

	return result.String()
}

// Analyze errors and provide intelligent recommendations
func (mcp *MCPServer) analyzeErrorsAndRecommend(jsonResponse string) string {
	var data struct {
		Logs []struct {
			Level   string `json:"level"`
			Service string `json:"service"`
			Message string `json:"message"`
		} `json:"logs"`
		Count int `json:"count"`
	}

	if err := json.Unmarshal([]byte(jsonResponse), &data); err != nil {
		return "Unable to analyze errors. Please check the logs manually."
	}

	if len(data.Logs) == 0 {
		return "✅ No errors found. Your system is healthy!"
	}

	// Categorize errors
	errorCategories := make(map[string][]string)
	serviceErrors := make(map[string]int)

	for _, log := range data.Logs {
		msg := strings.ToLower(log.Message)
		service := log.Service
		if service == "" {
			service = "unknown"
		}
		serviceErrors[service]++

		// Categorize by error type
		if strings.Contains(msg, "connection") || strings.Contains(msg, "refused") || strings.Contains(msg, "timeout") {
			errorCategories["connection"] = append(errorCategories["connection"], log.Message)
		} else if strings.Contains(msg, "permission") || strings.Contains(msg, "access denied") || strings.Contains(msg, "forbidden") {
			errorCategories["permission"] = append(errorCategories["permission"], log.Message)
		} else if strings.Contains(msg, "memory") || strings.Contains(msg, "heap") || strings.Contains(msg, "outofmemory") {
			errorCategories["memory"] = append(errorCategories["memory"], log.Message)
		} else if strings.Contains(msg, "certificate") || strings.Contains(msg, "ssl") || strings.Contains(msg, "tls") {
			errorCategories["certificate"] = append(errorCategories["certificate"], log.Message)
		} else if strings.Contains(msg, "413") || strings.Contains(msg, "entity too large") || strings.Contains(msg, "payload") {
			errorCategories["payload"] = append(errorCategories["payload"], log.Message)
		} else if strings.Contains(msg, "502") || strings.Contains(msg, "bad gateway") || strings.Contains(msg, "upstream") {
			errorCategories["upstream"] = append(errorCategories["upstream"], log.Message)
		} else if strings.Contains(msg, "circuit") || strings.Contains(msg, "breaker") {
			errorCategories["circuit"] = append(errorCategories["circuit"], log.Message)
		} else {
			errorCategories["other"] = append(errorCategories["other"], log.Message)
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("📊 **Analysis:** Found %d errors across %d service(s)\n\n", data.Count, len(serviceErrors)))

	// Service breakdown
	if len(serviceErrors) > 0 {
		result.WriteString("**Affected Services:**\n")
		for service, count := range serviceErrors {
			result.WriteString(fmt.Sprintf("• %s: %d error(s)\n", service, count))
		}
		result.WriteString("\n")
	}

	// Category-based recommendations
	result.WriteString("**Recommendations by Category:**\n\n")

	if len(errorCategories["connection"]) > 0 {
		result.WriteString("🔌 **Connection Issues** (" + fmt.Sprintf("%d", len(errorCategories["connection"])) + " errors):\n")
		result.WriteString("• Check network connectivity between services\n")
		result.WriteString("• Verify service endpoints and ports are correct\n")
		result.WriteString("• Review firewall rules and security groups\n")
		result.WriteString("• Check if target services are running and healthy\n\n")
	}

	if len(errorCategories["permission"]) > 0 {
		result.WriteString("🔐 **Permission/Access Issues** (" + fmt.Sprintf("%d", len(errorCategories["permission"])) + " errors):\n")
		result.WriteString("• Review IAM policies and access controls\n")
		result.WriteString("• Verify API keys and credentials are valid\n")
		result.WriteString("• Check S3 bucket policies and permissions\n")
		result.WriteString("• Ensure service accounts have proper roles\n\n")
	}

	if len(errorCategories["memory"]) > 0 {
		result.WriteString("💾 **Memory Issues** (" + fmt.Sprintf("%d", len(errorCategories["memory"])) + " errors):\n")
		result.WriteString("• Increase JVM heap size (-Xmx)\n")
		result.WriteString("• Review memory-intensive operations\n")
		result.WriteString("• Check for memory leaks in application code\n")
		result.WriteString("• Consider horizontal scaling or reducing load\n\n")
	}

	if len(errorCategories["certificate"]) > 0 {
		result.WriteString("🔒 **Certificate/SSL Issues** (" + fmt.Sprintf("%d", len(errorCategories["certificate"])) + " errors):\n")
		result.WriteString("• Verify SSL certificates are valid and not expired\n")
		result.WriteString("• Check certificate chain configuration\n")
		result.WriteString("• Review trust store configuration\n")
		result.WriteString("• Ensure proper certificate validation settings\n\n")
	}

	if len(errorCategories["payload"]) > 0 {
		result.WriteString("📦 **Payload Size Issues** (" + fmt.Sprintf("%d", len(errorCategories["payload"])) + " errors):\n")
		result.WriteString("• Increase client_max_body_size in Nginx\n")
		result.WriteString("• Review API request size limits\n")
		result.WriteString("• Consider implementing file upload limits\n")
		result.WriteString("• Use chunked uploads for large files\n\n")
	}

	if len(errorCategories["upstream"]) > 0 {
		result.WriteString("⬆️ **Upstream/Backend Issues** (" + fmt.Sprintf("%d", len(errorCategories["upstream"])) + " errors):\n")
		result.WriteString("• Check backend service health and availability\n")
		result.WriteString("• Review load balancer configuration\n")
		result.WriteString("• Verify backend endpoints are correct\n")
		result.WriteString("• Check for upstream timeout settings\n\n")
	}

	if len(errorCategories["circuit"]) > 0 {
		result.WriteString("⚡ **Circuit Breaker Issues** (" + fmt.Sprintf("%d", len(errorCategories["circuit"])) + " errors):\n")
		result.WriteString("• Review circuit breaker thresholds\n")
		result.WriteString("• Check dependency service health\n")
		result.WriteString("• Consider implementing retry logic with backoff\n")
		result.WriteString("• Monitor circuit breaker state transitions\n\n")
	}

	if len(errorCategories["other"]) > 0 {
		result.WriteString("📝 **Other Issues** (" + fmt.Sprintf("%d", len(errorCategories["other"])) + " errors):\n")
		result.WriteString("• Review error logs for specific patterns\n")
		result.WriteString("• Check application configuration\n")
		result.WriteString("• Verify dependencies and versions\n")
		result.WriteString("• Consider enabling more detailed logging\n\n")
	}

	result.WriteString("💡 **General Tips:**\n")
	result.WriteString("• Monitor error rates over time to identify trends\n")
	result.WriteString("• Set up alerts for critical error patterns\n")
	result.WriteString("• Review error logs during peak traffic periods\n")
	result.WriteString("• Consider implementing automated error recovery mechanisms\n")

	return result.String()
}

func main() {
	mcp := NewMCPServer()
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

	r.POST("/mcp/query", mcp.handleMCPQuery)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"llm_enabled": mcp.useLLM,
			"llm_provider": "gemini",
		})
	})

	log.Println("MCP Server listening on :5001")
	if err := r.Run(":5001"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
