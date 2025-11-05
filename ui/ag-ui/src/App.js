import React, { useState } from 'react';
import ChatSidebar from './components/ChatSidebar';
import LogPanel from './components/LogPanel';
import MetricsChart from './components/MetricsChart';
import StatsBar from './components/StatsBar';
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
        body: JSON.stringify({ query: query })
      });
      const data = await response.json();
      setLlmResponse(data.response || 'No response received');
      
      // PoC: A real app would parse this response and update logs/metrics
    } catch (error) {
      setLlmResponse(`Error: ${error.message}`);
    }
  };

  return (
    <div className="app-container">
      <ChatSidebar onQuery={handleQuery} response={llmResponse} />
      <div className="main-content">
        <StatsBar />
        <MetricsChart data={metrics} />
        <LogPanel logs={logs} />
      </div>
    </div>
  );
}

export default App;
