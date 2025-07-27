import React, { useState, useEffect } from 'react';
import { NavLink } from 'react-router-dom';
import { format } from 'date-fns';
import './Sidebar.css';

const Sidebar = () => {
  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  const menuItems = [
    {
      title: 'Data View',
      path: '/data',
      icon: '📊',
      description: 'View statistics and analytics'
    },
    {
      title: 'Earth View',
      path: '/earth',
      icon: '🌍',
      description: 'Global infrastructure map'
    },
    {
      title: 'Member View',
      path: '/members',
      icon: '👥',
      description: 'Member information and stats'
    },
    {
      title: 'Billing View',
      path: '/billing',
      icon: '💰',
      description: 'Billing management and PDFs'
    }
  ];

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <div className="logo-container">
          <img src="/static/imgs/ibp.png" alt="IBP" className="logo" />
          <h1 className="logo-text"> Dashboard</h1>
        </div>
      </div>
                     
      <nav className="sidebar-nav">
        {menuItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
          >
            <span className="nav-icon">{item.icon}</span>
            <div className="nav-content">
              <span className="nav-title">{item.title}</span>
              <span className="nav-description">{item.description}</span>
            </div>
          </NavLink>
        ))}
      </nav>

      <div className="sidebar-footer">
        <div className="time-display">
          <div className="date">{format(currentTime, 'EEEE, MMMM d, yyyy')}</div>
          <div className="time">{format(currentTime, 'HH:mm:ss')} UTC</div>
        </div>
        <div className="version">
          <small>IBP GeoDNS v0.4.0</small>
        </div>
      </div>
    </div>
  );
};

export default Sidebar;