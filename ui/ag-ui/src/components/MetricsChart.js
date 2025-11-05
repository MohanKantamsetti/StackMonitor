import React, { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import './MetricsChart.css';

function MetricsChart({ data: propData }) {
  const [metrics, setMetrics] = useState([]);

  useEffect(() => {
    // Fetch metrics for all log levels
    const fetchMetrics = async () => {
      try {
        // Fetch error rate
        const errorRes = await fetch('http://localhost:5000/api/v1/metrics/error-rate?range=1h');
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
    const interval = setInterval(fetchMetrics, 10000); // Refresh every 10s

    return () => clearInterval(interval);
  }, []);

  // Use metrics directly (already processed)
  const chartData = metrics;

  return (
    <div className="metrics-chart">
      <div className="metrics-header">
        <h3>Log Levels Over Time</h3>
      </div>
      <div className="metrics-content">
        {chartData.length === 0 ? (
          <div className="no-metrics">No metrics data available</div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis domain={[0, 'auto']} /> {/* Start from 0 to ground the chart */}
              <Tooltip />
              <Legend />
              <Line 
                type="monotone" 
                dataKey="errors" 
                stroke="#f44336" 
                strokeWidth={2} 
                name="Errors"
                dot={{ r: 3 }}
                isAnimationActive={true}
              />
              <Line 
                type="monotone" 
                dataKey="warnings" 
                stroke="#ff9800" 
                strokeWidth={2} 
                name="Warnings"
                dot={{ r: 3 }}
                isAnimationActive={true}
              />
              <Line 
                type="monotone" 
                dataKey="info" 
                stroke="#4caf50" 
                strokeWidth={2} 
                name="Info"
                dot={{ r: 3 }}
                isAnimationActive={true}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}

export default MetricsChart;
