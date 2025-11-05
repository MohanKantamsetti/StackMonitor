import React, { useState, useEffect } from 'react';
import './LogPanel.css';

function LogPanel({ logs: propLogs }) {
  const [logs, setLogs] = useState(propLogs || []);
  const [filter, setFilter] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);

  // Polling instead of WebSocket for reliability
  useEffect(() => {
    if (!autoRefresh) return;
    
    const fetchLogs = () => {
      fetch('http://localhost:5000/api/v1/logs?limit=100')
        .then(res => res.json())
        .then(data => {
          if (data && data.logs && Array.isArray(data.logs)) {
            setLogs(data.logs);
          } else {
            setLogs([]);
          }
        })
        .catch(err => {
          console.error('Error fetching logs:', err);
          setLogs([]);
        });
    };

    fetchLogs(); // Initial fetch
    const interval = setInterval(fetchLogs, 5000); // Poll every 5 seconds
    return () => clearInterval(interval);
  }, [autoRefresh]);

  const filteredLogs = logs.filter(log => {
    if (!filter) return true;
    const searchTerm = filter.toLowerCase();
    return (
      log.message?.toLowerCase().includes(searchTerm) ||
      log.service?.toLowerCase().includes(searchTerm) ||
      log.level?.toLowerCase().includes(searchTerm)
    );
  });

  const getLevelClass = (level) => {
    switch (level?.toUpperCase()) {
      case 'ERROR':
        return 'log-entry-error';
      case 'WARN':
        return 'log-entry-warn';
      case 'INFO':
        return 'log-entry-info';
      default:
        return '';
    }
  };

  return (
    <div className="log-panel">
      <div className="log-panel-header">
        <h3>Live Logs</h3>
        <div className="log-controls">
          <button 
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`refresh-toggle ${autoRefresh ? 'active' : ''}`}
          >
            {autoRefresh ? '⏸ Pause' : '▶ Resume'}
          </button>
          <button 
            onClick={() => {
              fetch('http://localhost:5000/api/v1/logs?limit=100')
                .then(res => res.json())
                .then(data => {
                  if (data && data.logs && Array.isArray(data.logs)) {
                    setLogs(data.logs);
                  }
                })
                .catch(err => console.error('Error fetching logs:', err));
            }}
            className="refresh-button"
          >
            🔄 Refresh
          </button>
          <input
            type="text"
            placeholder="Filter logs..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="log-filter"
          />
        </div>
      </div>
      <div className="log-entries">
        {filteredLogs.length === 0 ? (
          <div className="no-logs">No logs to display</div>
        ) : (
          filteredLogs.map((log, idx) => (
            <div key={idx} className={`log-entry ${getLevelClass(log.level)}`}>
              <span className="log-timestamp">{log.timestamp}</span>
              <span className={`log-level log-level-${log.level?.toLowerCase()}`}>
                {log.level}
              </span>
              <span className="log-service">[{log.service}]</span>
              <span className="log-message">{log.message}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

export default LogPanel;
