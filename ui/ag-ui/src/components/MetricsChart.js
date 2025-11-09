import React, { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import './MetricsChart.css';

function MetricsChart({ data: propData }) {
  const [metrics, setMetrics] = useState([]);
  const [timeRange, setTimeRange] = useState('1h'); // '15m', '1h', '6h', '24h'
  const [refreshInterval, setRefreshInterval] = useState(10000); // milliseconds
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [showErrors, setShowErrors] = useState(true);
  const [showWarnings, setShowWarnings] = useState(true);
  const [showInfo, setShowInfo] = useState(true);
  const [showDataPoints, setShowDataPoints] = useState(true);
  const [chartHeight, setChartHeight] = useState(300);

  useEffect(() => {
    if (!autoRefresh) return;

    // Fetch metrics for all log levels
    const fetchMetrics = async () => {
      try {
        // Fetch error rate
        const errorRes = await fetch(`http://localhost:5000/api/v1/metrics/error-rate?range=${timeRange}`);
        const errorData = await errorRes.json();
        
        // Fetch warning rate (using logs endpoint with aggregation)
        const warnRes = await fetch('http://localhost:5000/api/v1/logs?level=WARN&limit=100');
        const warnData = await warnRes.json();
        
        // Fetch info logs
        const infoRes = await fetch('http://localhost:5000/api/v1/logs?level=INFO&limit=100');
        const infoData = await infoRes.json();
        
        // Process error metrics
        const errorMetrics = errorData?.metrics || [];
        
        // Process warnings and info by time buckets
        const timeBuckets = {};
        
        // Group errors by time
        errorMetrics.forEach(m => {
          const timeKey = new Date(m.time).toISOString();
          if (!timeBuckets[timeKey]) {
            timeBuckets[timeKey] = { time: m.time, errors: 0, warnings: 0, info: 0 };
          }
          timeBuckets[timeKey].errors = Number(m.count) || 0;
        });
        
        // Group warnings by time (1 minute buckets)
        if (warnData?.logs) {
          warnData.logs.forEach(log => {
            const logTime = new Date(log.timestamp);
            const timeKey = new Date(logTime.getFullYear(), logTime.getMonth(), logTime.getDate(), 
              logTime.getHours(), logTime.getMinutes(), 0).toISOString();
            if (!timeBuckets[timeKey]) {
              timeBuckets[timeKey] = { time: timeKey, errors: 0, warnings: 0, info: 0 };
            }
            timeBuckets[timeKey].warnings++;
          });
        }
        
        // Group info by time (1 minute buckets)
        if (infoData?.logs) {
          infoData.logs.forEach(log => {
            const logTime = new Date(log.timestamp);
            const timeKey = new Date(logTime.getFullYear(), logTime.getMonth(), logTime.getDate(), 
              logTime.getHours(), logTime.getMinutes(), 0).toISOString();
            if (!timeBuckets[timeKey]) {
              timeBuckets[timeKey] = { time: timeKey, errors: 0, warnings: 0, info: 0 };
            }
            timeBuckets[timeKey].info++;
          });
        }
        
        // Convert to array and sort
        const metricsArray = Object.values(timeBuckets)
          .map(m => ({
            time: new Date(m.time).toLocaleTimeString(),
            errors: m.errors || 0,
            warnings: m.warnings || 0,
            info: m.info || 0
          }))
          .sort((a, b) => new Date(a.time) - new Date(b.time));
        
        setMetrics(metricsArray);
      } catch (err) {
        console.error('Error fetching metrics:', err);
        setMetrics([]);
      }
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, refreshInterval);

    return () => clearInterval(interval);
  }, [timeRange, refreshInterval, autoRefresh]);

  // Use metrics directly (already processed)
  const chartData = metrics;

  const manualRefresh = async () => {
    try {
      const errorRes = await fetch(`http://localhost:5000/api/v1/metrics/error-rate?range=${timeRange}`);
      const errorData = await errorRes.json();
      
      const warnRes = await fetch('http://localhost:5000/api/v1/logs?level=WARN&limit=100');
      const warnData = await warnRes.json();
      
      const infoRes = await fetch('http://localhost:5000/api/v1/logs?level=INFO&limit=100');
      const infoData = await infoRes.json();
      
      const errorMetrics = errorData?.metrics || [];
      const timeBuckets = {};
      
      errorMetrics.forEach(m => {
        const timeKey = new Date(m.time).toISOString();
        if (!timeBuckets[timeKey]) {
          timeBuckets[timeKey] = { time: m.time, errors: 0, warnings: 0, info: 0 };
        }
        timeBuckets[timeKey].errors = Number(m.count) || 0;
      });
      
      if (warnData?.logs) {
        warnData.logs.forEach(log => {
          const logTime = new Date(log.timestamp);
          const timeKey = new Date(logTime.getFullYear(), logTime.getMonth(), logTime.getDate(), 
            logTime.getHours(), logTime.getMinutes(), 0).toISOString();
          if (!timeBuckets[timeKey]) {
            timeBuckets[timeKey] = { time: timeKey, errors: 0, warnings: 0, info: 0 };
          }
          timeBuckets[timeKey].warnings++;
        });
      }
      
      if (infoData?.logs) {
        infoData.logs.forEach(log => {
          const logTime = new Date(log.timestamp);
          const timeKey = new Date(logTime.getFullYear(), logTime.getMonth(), logTime.getDate(), 
            logTime.getHours(), logTime.getMinutes(), 0).toISOString();
          if (!timeBuckets[timeKey]) {
            timeBuckets[timeKey] = { time: timeKey, errors: 0, warnings: 0, info: 0 };
          }
          timeBuckets[timeKey].info++;
        });
      }
      
      const metricsArray = Object.values(timeBuckets)
        .map(m => ({
          time: new Date(m.time).toLocaleTimeString(),
          errors: m.errors || 0,
          warnings: m.warnings || 0,
          info: m.info || 0
        }))
        .sort((a, b) => new Date(a.time) - new Date(b.time));
      
      setMetrics(metricsArray);
    } catch (err) {
      console.error('Error fetching metrics:', err);
    }
  };

  const cycleChartHeight = () => {
    const heights = [250, 300, 400, 500];
    const currentIndex = heights.indexOf(chartHeight);
    const nextIndex = (currentIndex + 1) % heights.length;
    setChartHeight(heights[nextIndex]);
  };

  const getHeightLabel = () => {
    const labels = { 250: '↕ S', 300: '↕ M', 400: '↕ L', 500: '↕ XL' };
    return labels[chartHeight] || '↕ M';
  };

  return (
    <div className="metrics-chart">
      <div className="metrics-header">
        <h3>Log Levels Over Time</h3>
        <div className="chart-controls">
          <select 
            value={timeRange} 
            onChange={(e) => setTimeRange(e.target.value)}
            className="chart-control-select"
            title="Time range"
          >
            <option value="15m">Last 15 min</option>
            <option value="1h">Last 1 hour</option>
            <option value="6h">Last 6 hours</option>
            <option value="24h">Last 24 hours</option>
            <option value="all">All (30 days)</option>
          </select>
          <button 
            onClick={cycleChartHeight}
            className="chart-control-button"
            title="Cycle chart height"
          >
            {getHeightLabel()}
          </button>
          <button 
            onClick={() => setShowErrors(!showErrors)}
            className={`chart-control-button ${showErrors ? 'active-red' : ''}`}
            title="Toggle error line"
          >
            {showErrors ? '🔴' : '⚫'} Errors
          </button>
          <button 
            onClick={() => setShowWarnings(!showWarnings)}
            className={`chart-control-button ${showWarnings ? 'active-orange' : ''}`}
            title="Toggle warning line"
          >
            {showWarnings ? '🟠' : '⚫'} Warnings
          </button>
          <button 
            onClick={() => setShowInfo(!showInfo)}
            className={`chart-control-button ${showInfo ? 'active-green' : ''}`}
            title="Toggle info line"
          >
            {showInfo ? '🟢' : '⚫'} Info
          </button>
          <button 
            onClick={() => setShowDataPoints(!showDataPoints)}
            className={`chart-control-button ${showDataPoints ? 'active' : ''}`}
            title="Toggle data points"
          >
            {showDataPoints ? '●' : '○'} Points
          </button>
          <button 
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`chart-control-button ${autoRefresh ? 'active' : ''}`}
            title={autoRefresh ? 'Pause auto-refresh' : 'Resume auto-refresh'}
          >
            {autoRefresh ? '⏸ Pause' : '▶ Resume'}
          </button>
          <button 
            onClick={manualRefresh}
            className="chart-control-button"
            title="Refresh now"
          >
            🔄 Refresh
          </button>
          <select 
            value={refreshInterval} 
            onChange={(e) => setRefreshInterval(Number(e.target.value))}
            className="chart-control-select"
            title="Auto-refresh interval"
          >
            <option value="5000">5s interval</option>
            <option value="10000">10s interval</option>
            <option value="30000">30s interval</option>
            <option value="60000">60s interval</option>
          </select>
        </div>
      </div>
      <div className="metrics-content">
        {chartData.length === 0 ? (
          <div className="no-metrics">No metrics data available</div>
        ) : (
          <ResponsiveContainer width="100%" height={chartHeight}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
              <XAxis 
                dataKey="time" 
                style={{ fontSize: '12px' }}
              />
              <YAxis 
                domain={[0, 'auto']} 
                style={{ fontSize: '12px' }}
              />
              <Tooltip 
                contentStyle={{ 
                  backgroundColor: 'rgba(255,255,255,0.95)', 
                  border: '1px solid #ccc',
                  borderRadius: '4px'
                }}
              />
              <Legend 
                wrapperStyle={{ paddingTop: '10px' }}
              />
              {showErrors && (
                <Line 
                  type="monotone" 
                  dataKey="errors" 
                  stroke="#f44336" 
                  strokeWidth={2} 
                  name="Errors"
                  dot={showDataPoints ? { r: 3 } : false}
                  isAnimationActive={true}
                  activeDot={{ r: 5 }}
                />
              )}
              {showWarnings && (
                <Line 
                  type="monotone" 
                  dataKey="warnings" 
                  stroke="#ff9800" 
                  strokeWidth={2} 
                  name="Warnings"
                  dot={showDataPoints ? { r: 3 } : false}
                  isAnimationActive={true}
                  activeDot={{ r: 5 }}
                />
              )}
              {showInfo && (
                <Line 
                  type="monotone" 
                  dataKey="info" 
                  stroke="#4caf50" 
                  strokeWidth={2} 
                  name="Info"
                  dot={showDataPoints ? { r: 3 } : false}
                  isAnimationActive={true}
                  activeDot={{ r: 5 }}
                />
              )}
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}

export default MetricsChart;
