import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Loading from '../components/Loading/Loading';
import MemberLogo from '../components/MemberLogo/MemberLogo';
import { getDownServices, getMemberHealth } from '../utils/common';
import { getMemberStatus } from '../utils/memberUtils';
import './MemberView.css';

const MemberView = () => {
  const [members, setMembers] = useState([]);
  const [downtime, setDowntime] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterLevel, setFilterLevel] = useState('all');
  const navigate = useNavigate();

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [membersRes, downtimeRes] = await Promise.all([
        ApiHelper.fetchMembers(),
        ApiHelper.fetchCurrentDowntime()
      ]);
      setMembers(membersRes.data);
      setDowntime(downtimeRes.data);
      setLoading(false);
    } catch (error) {
      console.error('Error loading data:', error);
      setLoading(false);
    }
  };

  const filteredMembers = members.filter(member => {
    const matchesSearch = member.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         member.region.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesLevel = filterLevel === 'all' || member.level === parseInt(filterLevel);
    return matchesSearch && matchesLevel;
  });

  const groupedMembers = filteredMembers.reduce((acc, member) => {
    const level = member.level || 1;
    if (!acc[level]) acc[level] = [];
    acc[level].push(member);
    return acc;
  }, {});

  if (loading) {
    return <Loading pageLevel={true} dataReady={true} />;
  }

  const renderMemberCard = (member) => {
    const status = getMemberStatus(member, downtime);
    const health = getMemberHealth(member, downtime);
    const downServices = getDownServices(member.name, member.services || [], downtime);
    
    return (
      <div
        key={member.name}
        className={`member-card ${status}`}
        onClick={() => navigate(`/members/${member.name}`)}
      >
        <div className="member-status-strip"></div>
        
        <a 
          href={member.website} 
          target="_blank" 
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()} 
          className="member-website-btn"
        >
          🌐
        </a>

        <div className="member-content">
          <div className="member-main-row">
            <div className="member-logo-wrapper">
              <div className="member-logo">
                <MemberLogo member={member} size="medium" />
              </div>
              <div className={`online-badge ${status}`}>
                {health.toFixed(0)}%
              </div>
            </div>
            
            <div className="member-header-info">
              <div className="member-name-row">
                <h3 className="member-name">{member.name}</h3>
                <span className="member-level-badge">Level {member.level}</span>
              </div>
              <div className="member-meta-info">
                <div className="meta-item">
                  <span>📍</span>
                  <span>{member.region}</span>
                </div>
                <div className="meta-item">
                  <span>📅</span>
                  <span>{member.joined_date}</span>
                </div>
              </div>
            </div>
          </div>

          <div className="member-stats-row">
            <div className="stat-block">
              <div className="stat-block-value">{member.services?.length || 0}</div>
              <div className="stat-block-label">Services</div>
            </div>
            <div className="stat-block">
              <div className="stat-block-value">{member.latitude?.toFixed(0)}°</div>
              <div className="stat-block-label">Latitude</div>
            </div>
            <div className="stat-block">
              <div className="stat-block-value">{member.longitude?.toFixed(0)}°</div>
              <div className="stat-block-label">Longitude</div>
            </div>
          </div>
        </div>

        {downServices.size > 0 && (
          <div className="member-issues-footer">
            <span className="issues-icon">⚠️</span>
            <span>{downServices.size} of {member.services?.length || 0} services currently affected</span>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="member-view fade-in">
      <div className="view-header">
        <h1>IBP Members</h1>
        <div className="header-stats">
          <div className="stat-item">
            <span className="stat-value">{members.length}</span>
            <span className="stat-label">Total Members</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{members.filter(m => getMemberStatus(m, downtime) === 'operational').length}</span>
            <span className="stat-label">Operational</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{members.filter(m => m.services?.length > 0).reduce((sum, m) => sum + (m.services?.length || 0), 0)}</span>
            <span className="stat-label">Total Services</span>
          </div>
        </div>
      </div>

      <div className="member-controls">
        <div className="search-box">
          <span className="search-icon">🔍</span>
          <input
            type="text"
            placeholder="Search members by name or region..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="search-input"
          />
        </div>
        <select
          value={filterLevel}
          onChange={(e) => setFilterLevel(e.target.value)}
          className="filter-select"
        >
          <option value="all">All Levels</option>
          <option value="3">Level 3</option>
          <option value="4">Level 4</option>
          <option value="5">Level 5</option>
          <option value="6">Level 6</option>
          <option value="7">Level 7</option>
        </select>
      </div>

      <div className="members-container">
        {/* Level 6 Members First */}
        {groupedMembers[6] && (
          <div className="level-section">
            <h2 className="level-header">Level 6 Members</h2>
            <div className="members-list">
              {groupedMembers[6].map(member => renderMemberCard(member))}
            </div>
          </div>
        )}

        {/* Level 5 Members */}
        {groupedMembers[5] && (
          <div className="level-section">
            <h2 className="level-header">Level 5 Members</h2>
            <div className="members-list">
              {groupedMembers[5].map(member => renderMemberCard(member))}
            </div>
          </div>
        )}

        {/* Level 3 Members */}
        {groupedMembers[3] && (
          <div className="level-section">
            <h2 className="level-header">Level 3 Members</h2>
            <div className="members-list">
              {groupedMembers[3].map(member => renderMemberCard(member))}
            </div>
          </div>
        )}

        {/* Other levels if they exist */}
        {Object.entries(groupedMembers)
          .filter(([level]) => !['3', '5', '6'].includes(level))
          .sort((a, b) => b[0] - a[0])
          .map(([level, levelMembers]) => (
            <div key={level} className="level-section">
              <h2 className="level-header">Level {level} Members</h2>
              <div className="members-list">
                {levelMembers.map(member => renderMemberCard(member))}
              </div>
            </div>
          ))}
      </div>
    </div>
  );
};

export default MemberView;