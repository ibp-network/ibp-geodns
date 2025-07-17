import React from 'react';
import { NavLink } from 'react-router-dom';
import './Sidebar.css';

const Sidebar = () => {
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
      title: 'Members',
      path: '/members',
      icon: '👥',
      description: 'Member information and stats'
    }
  ];

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <div className="logo-container">
          <img src="/static/imgs/ibp.png" alt="IBP" className="logo" />
          <h1 className="logo-text">IBP Dashboard</h1>
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
        <div className="version">
          Infrastructure Builders Program
          <br />
          <small>v0.4.0</small>
        </div>
      </div>
    </div>
  );
};

export default Sidebar;