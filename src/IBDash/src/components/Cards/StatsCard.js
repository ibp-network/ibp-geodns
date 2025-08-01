import React from 'react';
import './StatsCard.css';

const StatsCard = ({ title, value, subtitle, icon, trend, color }) => {
  const getColorClass = () => {
    switch (color) {
      case 'primary':
        return 'stats-card-primary';
      case 'success':
        return 'stats-card-success';
      case 'warning':
        return 'stats-card-warning';
      case 'error':
        return 'stats-card-error';
      default:
        return '';
    }
  };

  return (
    <div className={`stats-card ${getColorClass()}`}>
      <div className="stats-card-header">
        <h3 className="stats-card-title">{title}</h3>
        {icon && <div className="stats-card-icon">{icon}</div>}
      </div>
      
      <div className="stats-card-value">
        {value}
        {trend && (
          <span className={`stats-trend ${trend > 0 ? 'trend-up' : 'trend-down'}`}>
            {trend > 0 ? '↑' : '↓'} {Math.abs(trend)}%
          </span>
        )}
      </div>
      
      {subtitle && <p className="stats-card-subtitle">{subtitle}</p>}
    </div>
  );
};

export default StatsCard;