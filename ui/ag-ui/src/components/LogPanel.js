import React, { useState, useEffect, useRef } from 'react';
import './LogPanel.css';

function LogPanel({ query, logData }) {
  const [logs, setLogs] = useState([]);
  const [filter, setFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [showQueryResults, setShowQueryResults] = useState(false);
  const intervalRef = useRef(null);

  const fetchLogs = async () => {
    try {
      const response = await fetch('/api/v1/logs?limit=100');
      if (response.ok) {
        const data = await response.json();
        setLogs(data);
        setLoading(false);
      }
    } catch (error) {
      console.error('Error fetching logs:', error);
      setLoading(false);
    }
  };

  useEffect(() => {
    // Initial fetch
    fetchLogs();

    // Set up auto-refresh
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchLogs, 5000); // Refresh every 5 seconds
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [autoRefresh]);

  // Handle query results from AI
  useEffect(() => {
    if (logData && Array.isArray(logData) && logData.length > 0) {
      setLogs(logData);
      setShowQueryResults(true);
      setAutoRefresh(false); // Pause auto-refresh when showing query results
      setFilter(''); // Clear filter to show all query results
    }
  }, [logData]);

  useEffect(() => {
    if (query) {
      // Extract meaningful filter terms from query
      const lowerQuery = query.toLowerCase();
      if (lowerQuery.includes('error')) {
        setFilter('error');
      } else if (lowerQuery.includes('warn')) {
        setFilter('warn');
      } else if (lowerQuery.includes('info')) {
        setFilter('info');
      } else {
        setFilter('');
      }
    }
  }, [query]);

  const filteredLogs = logs.filter(log => {
    if (!filter) return true;
    const searchStr = filter.toLowerCase();
    return (
      log.message?.toLowerCase().includes(searchStr) ||
      log.level?.toLowerCase().includes(searchStr) ||
      log.source?.toLowerCase().includes(searchStr)
    );
  });

  const getLevelClass = (level) => {
    switch (level?.toUpperCase()) {
      case 'ERROR': return 'log-error';
      case 'WARN': return 'log-warn';
      case 'INFO': return 'log-info';
      case 'DEBUG': return 'log-debug';
      default: return '';
    }
  };

  const getLevelIcon = (level) => {
    switch (level?.toUpperCase()) {
      case 'ERROR': return '🔴';
      case 'WARN': return '⚠️';
      case 'INFO': return 'ℹ️';
      case 'DEBUG': return '🔧';
      default: return '📝';
    }
  };

  const formatTimestamp = (timestamp) => {
    try {
      const date = new Date(timestamp);
      return date.toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      });
    } catch (e) {
      return timestamp;
    }
  };

  const handleBackToLive = () => {
    setShowQueryResults(false);
    setAutoRefresh(true);
    fetchLogs();
  };

  return (
    <div className="log-panel">
      <div className="log-controls">
        <div className="log-controls-left">
          <input
            type="text"
            placeholder="🔍 Filter logs by level, source, or message..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="log-filter"
          />
        </div>
        <div className="log-controls-right">
          {showQueryResults && (
            <button onClick={handleBackToLive} className="back-to-live-button">
              ← Back to Live Logs
            </button>
          )}
          <label className="auto-refresh-toggle">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            <span>Auto-refresh</span>
          </label>
          <button onClick={fetchLogs} className="refresh-button" title="Refresh now">
            🔄
          </button>
          <span className="log-count">
            {filteredLogs.length} / {logs.length} logs
            {showQueryResults && <span className="query-badge">Query Results</span>}
          </span>
        </div>
      </div>

      <div className="log-list">
        {loading ? (
          <div className="log-empty">
            <div className="loading-spinner"></div>
            <p>Loading logs...</p>
          </div>
        ) : filteredLogs.length === 0 ? (
          <div className="log-empty">
            {logs.length === 0 ? (
              <>
                <span style={{fontSize: '3rem'}}>📭</span>
                <p>No logs available yet</p>
                <small>Logs will appear here as they are generated</small>
              </>
            ) : (
              <>
                <span style={{fontSize: '3rem'}}>🔍</span>
                <p>No logs match your filter</p>
                <small>Try adjusting your search criteria</small>
              </>
            )}
          </div>
        ) : (
          filteredLogs.map((log, idx) => (
            <div key={idx} className={`log-entry ${getLevelClass(log.level)}`}>
              <span className="log-icon">{getLevelIcon(log.level)}</span>
              <span className="log-timestamp" title={log.timestamp}>
                {formatTimestamp(log.timestamp)}
              </span>
              <span className="log-level">{log.level}</span>
              <span className="log-source" title={`Agent: ${log.agent_id || 'unknown'}`}>
                [{log.source}]
              </span>
              <span className="log-message">{log.message}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

export default LogPanel;
