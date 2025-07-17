import React from 'react';
import './Loading.css';

const Loading = () => {
  return (
    <div className="loading-screen">
      <div className="loading-content">
        <img src="/static/imgs/ibp.png" alt="IBP" className="loading-logo" />
        <div className="loading-spinner"></div>
        <h2>Infrastructure Builders Program</h2>
        <p>Loading dashboard...</p>
      </div>
    </div>
  );
};

export default Loading;