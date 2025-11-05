import React, { useState, useEffect } from 'react';
import './StatsBar.css';

function StatsBar() {
  const [stats, setStats] = useState({
    total: 0,
    errors: 0,
    warnings: 0,
    info: 0
  });

  useEffect(() => {
    const fetchStats = () => {
      fetch('http://localhost:5000/api/v1/logs/stats')
        .then(res => res.json())
        .then(data => {
          if (data) {
            setStats({
              total: data.total || 0,
              errors: data.errors || 0,
              warnings: data.warnings || 0,
              info: data.info || 0
            });
          }
        })
        .catch(err => {
          console.error('Error fetching stats:', err);
          setStats({ total: 0, errors: 0, warnings: 0, info: 0 });
        });
    };

    fetchStats();
    const interval = setInterval(fetchStats, 5000); // Refresh every 5s
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="stats-bar">
      <div className="stat-item">
        <div className="stat-label">Total Logs</div>
        <div className="stat-value">{stats.total.toLocaleString()}</div>
      </div>
      <div className="stat-item stat-error">
        <div className="stat-label">Errors</div>
        <div className="stat-value">{stats.errors.toLocaleString()}</div>
      </div>
      <div className="stat-item stat-warn">
        <div className="stat-label">Warnings</div>
        <div className="stat-value">{stats.warnings.toLocaleString()}</div>
      </div>
      <div className="stat-item stat-info">
        <div className="stat-label">Info</div>
        <div className="stat-value">{stats.info.toLocaleString()}</div>
      </div>
    </div>
  );
}

export default StatsBar;

