import React from 'react';
import { NavLink } from 'react-router-dom';
import './Sidebar.css';

const Sidebar = ({ open, onToggle }) => {
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
    <div className={`sidebar ${open ? '' : 'closed'}`}>
      <div className="sidebar-header">
        <div className="logo-container">
          <img src="/ibp-logo.png" alt="IBP" className="logo" />
          {open && <h1 className="logo-text">IBP Dashboard</h1>}
        </div>
        <button className="toggle-btn" onClick={onToggle}>
          {open ? '◀' : '▶'}
        </button>
      </div>
      
      <nav className="sidebar-nav">
        {menuItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
          >
            <span className="nav-icon">{item.icon}</span>
            {open && (
              <div className="nav-content">
                <span className="nav-title">{item.title}</span>
                <span className="nav-description">{item.description}</span>
              </div>
            )}
          </NavLink>
        ))}
      </nav>

      {open && (
        <div className="sidebar-footer">
          <div className="version">
            Infrastructure Builders Program
            <br />
            <small>v0.4.0</small>
          </div>
        </div>
      )}
    </div>
  );
};

export default Sidebar;