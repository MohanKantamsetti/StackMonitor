import React, { useState } from 'react';
import ChatSidebar from './components/ChatSidebar';
import LogPanel from './components/LogPanel';
import MetricsChart from './components/MetricsChart';
import StatsBar from './components/StatsBar';
import ErrorBoundary from './components/ErrorBoundary';
import './App.css';

function App() {
  const [logs, setLogs] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [llmResponse, setLlmResponse] = useState("");

  // Handler for LLM queries
  const handleQuery = async (query) => {
    // Clear previous response to avoid stale data
    setLlmResponse('');
    
    // This calls the MCP server
    try {
      const response = await fetch('http://localhost:5001/mcp/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: query }),
        signal: AbortSignal.timeout(30000) // 30 second timeout
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      const data = await response.json();
      setLlmResponse(data.response || 'No response received');
      
      // PoC: A real app would parse this response and update logs/metrics
    } catch (error) {
      console.error('Query error:', error);
      if (error.name === 'AbortError' || error.name === 'TimeoutError') {
        setLlmResponse('⏱️ Request timed out. The MCP server took too long to respond. Please try a simpler query.');
      } else if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
        setLlmResponse('🔌 Cannot connect to MCP server. Please ensure the service is running.');
      } else {
        setLlmResponse(`❌ Error: ${error.message}`);
      }
    }
  };

  return (
    <div className="app-container">
      <ErrorBoundary>
        <ChatSidebar onQuery={handleQuery} response={llmResponse} />
      </ErrorBoundary>
      <div className="main-content">
        <ErrorBoundary>
          <StatsBar />
        </ErrorBoundary>
        <ErrorBoundary>
          <MetricsChart data={metrics} />
        </ErrorBoundary>
        <ErrorBoundary>
          <LogPanel logs={logs} />
        </ErrorBoundary>
      </div>
    </div>
  );
}

export default App;
