import React from 'react';
import { getStatusIcon } from '../../utils/common';
import './StatusBadge.css';

const StatusBadge = ({ status, value, type = 'default' }) => {
  const getText = () => {
    if (type === 'uptime' && value !== undefined) {
      return status === 'operational' ? 'Operational' : `${value.toFixed(0)}% Online`;
    }
    return status === 'operational' ? 'Operational' :
           status === 'degraded' ? 'Degraded' : 'Offline';
  };

  return (
    <div className={`status-badge status-${status}`}>
      <span className="status-icon">{getStatusIcon(status)}</span>
      <span className="status-text">{getText()}</span>
    </div>
  );
};

export default StatusBadge;