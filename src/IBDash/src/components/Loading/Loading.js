import React, { useState } from 'react';
import './Loading.css';

const Loading = () => {
  const [animationError, setAnimationError] = useState(false);

  // Using the ibp.gif from assets
  const loadingAnimation = '/static/imgs/ibp.gif';

  return (
    <div className="loading-screen">
      <div className="loading-content">
        {!animationError ? (
          <div className="loading-animation pulse">
            <img 
              src={loadingAnimation} 
              alt="Loading" 
              onError={() => setAnimationError(true)}
            />
          </div>
        ) : (
          // Fallback to original loading animation if gif fails
          <>
            <img src="/static/imgs/ibp.png" alt="IBP" className="loading-logo" />
            <div className="loading-spinner"></div>
          </>
        )}
        <h2 className="loading-title">Infrastructure Builders Program</h2>
        <p className="loading-subtitle">Loading dashboard...</p>
      </div>
    </div>
  );
};

export default Loading;