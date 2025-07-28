import React, { useState, useEffect, useRef } from 'react';
import './Loading.css';

const Loading = ({ onAnimationComplete, dataReady, pageLevel = false, minDuration = 3000 }) => {
  const [animationError, setAnimationError] = useState(false);
  const [progress, setProgress] = useState(0);
  const animationRef = useRef(null);
  const startTimeRef = useRef(null);
  const animationFrameRef = useRef(null);

  // GIF animation duration in milliseconds
  const GIF_DURATION = pageLevel ? 100 : minDuration; // Shorter duration for page-level loading

  useEffect(() => {
    if (!animationError) {
      startTimeRef.current = Date.now();
      
      // Update progress bar
      const updateProgress = () => {
        const elapsed = Date.now() - startTimeRef.current;
        const currentProgress = Math.min((elapsed / GIF_DURATION) * 100, 100);
        setProgress(currentProgress);

        if (currentProgress < 100) {
          animationFrameRef.current = requestAnimationFrame(updateProgress);
        } else {
          // Animation complete
          if (dataReady || pageLevel) {
            setTimeout(() => {
              if (onAnimationComplete) onAnimationComplete();
            }, 200); // Small delay for smooth transition
          }
        }
      };

      animationFrameRef.current = requestAnimationFrame(updateProgress);

      return () => {
        if (animationFrameRef.current) {
          cancelAnimationFrame(animationFrameRef.current);
        }
      };
    }
  }, [animationError, dataReady, onAnimationComplete, GIF_DURATION, pageLevel]);

  // Auto-proceed when data becomes ready after animation completes
  useEffect(() => {
    if (dataReady && progress >= 100) {
      setTimeout(() => {
        if (onAnimationComplete) onAnimationComplete();
      }, 200);
    }
  }, [dataReady, progress, onAnimationComplete]);

  const loadingAnimation = '/static/imgs/ibp.gif';

  const content = (
    <>
      {!animationError ? (
        <>
          <div className={`loading-animation ${pageLevel ? 'page-level' : ''}`} ref={animationRef}>
            <img 
              src={loadingAnimation} 
              alt="Loading" 
              onError={() => setAnimationError(true)}
            />
          </div>
          <h2 className="loading-title">Infrastructure Builders Program</h2>
          <p className="loading-subtitle">
            {pageLevel ? 'Downloading API Data...' : 'Loading dashboard...'}
          </p>
          <div className="loading-progress">
            <div 
              className="loading-progress-bar" 
              style={{ width: `${progress}%` }}
            />
          </div>
          {progress >= 100 && !dataReady && !pageLevel && (
            <p className="loading-waiting">Waiting for data...</p>
          )}
        </>
      ) : (
        // Fallback to original loading animation if gif fails
        <>
          <img src="/static/imgs/ibp.png" alt="IBP" className="loading-logo" />
          <div className="loading-spinner"></div>
          <h2 className="loading-title">Infrastructure Builders Program</h2>
          <p className="loading-subtitle">
            {pageLevel ? 'Downloading API Data...' : 'Loading...'}
          </p>
          <div className="loading-progress">
            <div 
              className="loading-progress-bar" 
              style={{ width: `${progress}%` }}
            />
          </div>
        </>
      )}
    </>
  );

  if (pageLevel) {
    return (
      <div className="page-loading-container fade-in">
        <div className="loading-content">
          {content}
        </div>
      </div>
    );
  }

  return (
    <div className="loading-screen">
      <div className="loading-content">
        {content}
      </div>
    </div>
  );
};

export default Loading;