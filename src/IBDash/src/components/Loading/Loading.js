import React, { useState, useEffect, useRef } from 'react';
import './Loading.css';

const Loading = ({ onAnimationComplete, dataReady, pageLevel = false, minDuration = 3000 }) => {
  const [animationError, setAnimationError] = useState(false);
  const [progress, setProgress] = useState(0);
  const animationRef = useRef(null);
  const startTimeRef = useRef(null);
  const animationFrameRef = useRef(null);
  const completedRef = useRef(false);

  // Minimum display time to avoid flashing
  const MIN_DISPLAY_TIME = 100;

  useEffect(() => {
    if (!animationError) {
      startTimeRef.current = Date.now();
      completedRef.current = false;
      
      // Update progress bar
      const updateProgress = () => {
        const elapsed = Date.now() - startTimeRef.current;
        
        if (dataReady && elapsed >= MIN_DISPLAY_TIME) {
          // If data is ready and minimum time has passed, complete immediately
          setProgress(100);
          if (!completedRef.current) {
            completedRef.current = true;
            setTimeout(() => {
              if (onAnimationComplete) onAnimationComplete();
            }, 100);
          }
        } else if (!dataReady) {
          // If data is not ready, show indeterminate progress
          const fakeProgress = Math.min((elapsed / minDuration) * 90, 90); // Cap at 90%
          setProgress(fakeProgress);
          animationFrameRef.current = requestAnimationFrame(updateProgress);
        } else {
          // Data is ready but minimum time hasn't passed
          const remainingTime = MIN_DISPLAY_TIME - elapsed;
          const currentProgress = Math.min((elapsed / MIN_DISPLAY_TIME) * 100, 100);
          setProgress(currentProgress);
          
          if (currentProgress < 100) {
            animationFrameRef.current = requestAnimationFrame(updateProgress);
          } else if (!completedRef.current) {
            completedRef.current = true;
            setTimeout(() => {
              if (onAnimationComplete) onAnimationComplete();
            }, 100);
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
  }, [animationError, dataReady, onAnimationComplete, minDuration]);

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
            {progress < 90 || !dataReady ? 'Downloading API Data...' : 'Almost ready...'}
          </p>
          <div className="loading-progress">
            <div 
              className="loading-progress-bar" 
              style={{ width: `${progress}%` }}
            />
          </div>
        </>
      ) : (
        // Fallback to original loading animation if gif fails
        <>
          <img src="/static/imgs/ibp.png" alt="IBP" className="loading-logo" />
          <div className="loading-spinner"></div>
          <h2 className="loading-title">Infrastructure Builders Program</h2>
          <p className="loading-subtitle">
            {progress < 90 || !dataReady ? 'Downloading API Data...' : 'Almost ready...'}
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