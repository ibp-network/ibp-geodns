import React, { useState, useEffect } from 'react';
import { format } from 'date-fns';
import './Header.css';

const Header = () => {
  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  return (
    <header className="header">
      <div className="header-left">
        <h2 className="page-title">Infrastructure Builders Dashboard</h2>
      </div>
      
      <div className="header-right">
        <div className="time-display">
          <div className="date">{format(currentTime, 'EEEE, MMMM d, yyyy')}</div>
          <div className="time">{format(currentTime, 'HH:mm:ss')} UTC</div>
        </div>
      </div>
    </header>
  );
};

export default Header;