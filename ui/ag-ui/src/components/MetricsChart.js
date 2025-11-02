import React, { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import './MetricsChart.css';

function MetricsChart() {
  const [metrics, setMetrics] = useState([]);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const response = await fetch('/api/metrics');
        const data = await response.json();
        setMetrics(data.metrics || []);
      } catch (error) {
        console.error('Error fetching metrics:', error);
      }
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 10000);
    return () => clearInterval(interval);
  }, []);

  // Generate sample data if no metrics available
  const chartData = metrics.length > 0 ? metrics : Array.from({ length: 10 }, (_, i) => ({
    time: new Date(Date.now() - (9 - i) * 60000).toLocaleTimeString(),
    errors: Math.floor(Math.random() * 20),
    warnings: Math.floor(Math.random() * 50),
    info: Math.floor(Math.random() * 100)
  }));

  return (
    <div className="metrics-chart">
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" stroke="#2a4365" />
          <XAxis dataKey="time" stroke="#a0aec0" />
          <YAxis stroke="#a0aec0" />
          <Tooltip 
            contentStyle={{ background: '#16213e', border: '1px solid #2a4365', borderRadius: '6px' }}
            labelStyle={{ color: '#e2e8f0' }}
          />
          <Legend />
          <Line type="monotone" dataKey="errors" stroke="#fc5c65" strokeWidth={2} />
          <Line type="monotone" dataKey="warnings" stroke="#fed330" strokeWidth={2} />
          <Line type="monotone" dataKey="info" stroke="#45aaf2" strokeWidth={2} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

export default MetricsChart;

