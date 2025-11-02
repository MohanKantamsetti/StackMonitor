import React, { useState, useEffect } from 'react';
import ChatSidebar from './components/ChatSidebar';
import LogPanel from './components/LogPanel';
import MetricsChart from './components/MetricsChart';
import './App.css';

function App() {
  const [query, setQuery] = useState('');
  const [logData, setLogData] = useState(null);
  const [stats, setStats] = useState({ total: 0, errors: 0, warns: 0, infos: 0 });

  const handleQuerySubmit = (userQuery, responseData) => {
    setQuery(userQuery);
    setLogData(responseData);
  };

  // Fetch stats periodically
  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await fetch('/api/v1/logs/stats');
        if (response.ok) {
          const data = await response.json();
          setStats(data);
        }
      } catch (error) {
        console.error('Error fetching stats:', error);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 10000); // Update every 10s
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="App">
      <header className="App-header">
        <div className="header-left">
          <h1>📊 StackMonitor</h1>
          <p>Intelligent Log Monitoring & Analysis</p>
        </div>
        <div className="header-stats">
          <div className="stat-card">
            <span className="stat-label">Total Logs</span>
            <span className="stat-value">{stats.total.toLocaleString()}</span>
          </div>
          <div className="stat-card errors">
            <span className="stat-label">Errors</span>
            <span className="stat-value">{stats.errors}</span>
          </div>
          <div className="stat-card warns">
            <span className="stat-label">Warnings</span>
            <span className="stat-value">{stats.warns}</span>
          </div>
          <div className="stat-card infos">
            <span className="stat-label">Info</span>
            <span className="stat-value">{stats.infos}</span>
          </div>
        </div>
      </header>
      
      <div className="main-container">
        <ChatSidebar onQuerySubmit={handleQuerySubmit} />
        
        <div className="content-area">
          <section className="metrics-section">
            <h2>📈 Error Rate Trends</h2>
            <MetricsChart />
          </section>
          
          <section className="logs-section">
            <h2>🔍 Live Logs</h2>
            <LogPanel query={query} logData={logData} />
          </section>
        </div>
      </div>
    </div>
  );
}

export default App;
