import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Charts from '../components/Charts/Charts';
import './MemberDetail.css';

const MemberDetail = () => {
  const { memberName } = useParams();
  const navigate = useNavigate();
  const [member, setMember] = useState(null);
  const [stats, setStats] = useState(null);
  const [billing, setBilling] = useState(null);
  const [downtime, setDowntime] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('overview');
  const [dateRange, setDateRange] = useState({
    start: new Date(new Date().setDate(new Date().getDate() - 30)),
    end: new Date()
  });

  useEffect(() => {
    loadMemberData();
  }, [memberName, dateRange]);

  const loadMemberData = async () => {
    try {
      const params = {
        start: dateRange.start.toISOString().split('T')[0],
        end: dateRange.end.toISOString().split('T')[0]
      };

      const [membersRes, statsRes, billingRes, downtimeRes] = await Promise.all([
        ApiHelper.fetchMembers(),
        ApiHelper.fetchMemberStats(memberName, params),
        ApiHelper.fetchBillingBreakdown({ member: memberName, include_downtime: true }),
        ApiHelper.fetchDowntimeEvents({ member: memberName, ...params })
      ]);

      const memberData = membersRes.data.find(m => m.name === memberName);
      setMember(memberData);
      setStats(statsRes.data);
      setBilling(billingRes.data);
      setDowntime(downtimeRes.data);
      setLoading(false);
    } catch (error) {
      console.error('Error loading member data:', error);
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="member-detail-loading">
        <div className="loading-spinner"></div>
        <p>Loading member details...</p>
      </div>
    );
  }

  if (!member) {
    return (
      <div className="member-not-found">
        <h2>Member not found</h2>
        <button onClick={() => navigate('/members')} className="btn btn-primary">
          Back to Members
        </button>
      </div>
    );
  }

  const tabs = [
    { id: 'overview', label: 'Overview', icon: '📊' },
    { id: 'billing', label: 'Billing', icon: '💰' },
    { id: 'downtime', label: 'Downtime', icon: '⚠️' },
    { id: 'usage', label: 'Usage Stats', icon: '📈' }
  ];

  return (
    <div className="member-detail fade-in">
      <div className="detail-header">
        <button onClick={() => navigate('/members')} className="back-button">
          ← Back to Members
        </button>
        <div className="member-title">
          {member.logo && (
            <img src={member.logo} alt={member.name} className="member-logo-large" />
          )}
          <div>
            <h1>{member.name}</h1>
            <p className="member-subtitle">Level {member.level} Member • {member.region}</p>
          </div>
        </div>
      </div>

      <div className="detail-stats">
        <div className="stat-card glass">
          <div className="stat-icon">📡</div>
          <div className="stat-content">
            <div className="stat-value">{stats?.total_requests?.toLocaleString() || '0'}</div>
            <div className="stat-label">Total Requests</div>
          </div>
        </div>
        <div className="stat-card glass">
          <div className="stat-icon">⚡</div>
          <div className="stat-content">
            <div className="stat-value">{member.services?.length || 0}</div>
            <div className="stat-label">Active Services</div>
          </div>
        </div>
        <div className="stat-card glass">
          <div className="stat-icon">🌐</div>
          <div className="stat-content">
            <div className="stat-value">{stats?.uptime_percentage?.toFixed(2) || '100'}%</div>
            <div className="stat-label">Uptime</div>
          </div>
        </div>
        <div className="stat-card glass">
          <div className="stat-icon">📅</div>
          <div className="stat-content">
            <div className="stat-value">{member.joined_date}</div>
            <div className="stat-label">Member Since</div>
          </div>
        </div>
      </div>

      <div className="detail-tabs card">
        <div className="tab-header">
          {tabs.map(tab => (
            <button
              key={tab.id}
              className={`tab-button ${activeTab === tab.id ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.id)}
            >
              <span className="tab-icon">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </div>

        <div className="tab-content">
          {activeTab === 'overview' && (
            <div className="overview-content">
              <div className="info-section">
                <h3>Member Information</h3>
                <div className="info-grid">
                  <div className="info-item">
                    <span className="info-label">Website</span>
                    <a href={member.website} target="_blank" rel="noopener noreferrer" className="info-value link">
                      {member.website}
                    </a>
                  </div>
                  <div className="info-item">
                    <span className="info-label">IPv4 Address</span>
                    <span className="info-value">{member.service_ipv4 || 'Not configured'}</span>
                  </div>
                  <div className="info-item">
                    <span className="info-label">IPv6 Address</span>
                    <span className="info-value">{member.service_ipv6 || 'Not configured'}</span>
                  </div>
                  <div className="info-item">
                    <span className="info-label">Location</span>
                    <span className="info-value">{member.latitude?.toFixed(4)}, {member.longitude?.toFixed(4)}</span>
                  </div>
                </div>
              </div>

              <div className="services-section">
                <h3>Active Services</h3>
                <div className="services-grid">
                  {member.services?.map(service => (
                    <div key={service} className="service-card">
                      <span className="service-icon">⚡</span>
                      <span className="service-name">{service}</span>
                    </div>
                  ))}
                </div>
              </div>

              {stats?.top_countries && (
                <div className="countries-section">
                  <h3>Top Countries by Requests</h3>
                  <Charts 
                    data={stats.top_countries.map(c => ({
                      country: c.country,
                      country_name: c.name,
                      requests: c.requests
                    }))} 
                    type="country" 
                  />
                </div>
              )}
            </div>
          )}

          {activeTab === 'billing' && billing && (
            <div className="billing-content">
              <h3>Current Month Billing</h3>
              {billing.members?.map(memberBilling => (
                <div key={memberBilling.name} className="billing-section">
                  <div className="billing-summary">
                    <div className="billing-item">
                      <span className="billing-label">Base Cost</span>
                      <span className="billing-value">${memberBilling.total_base_cost?.toFixed(2)}</span>
                    </div>
                    <div className="billing-item">
                      <span className="billing-label">Billed Amount</span>
                      <span className="billing-value highlight">${memberBilling.total_billed?.toFixed(2)}</span>
                    </div>
                    <div className="billing-item">
                      <span className="billing-label">SLA Credits</span>
                      <span className="billing-value credits">${memberBilling.total_credits?.toFixed(2)}</span>
                    </div>
                  </div>

                  <div className="services-billing">
                    <h4>Service Breakdown</h4>
                    <table className="billing-table">
                      <thead>
                        <tr>
                          <th>Service</th>
                          <th>Base Cost</th>
                          <th>Uptime</th>
                          <th>Billed</th>
                          <th>Credits</th>
                          <th>SLA Status</th>
                        </tr>
                      </thead>
                      <tbody>
                        {memberBilling.services?.map(service => (
                          <tr key={service.name}>
                            <td>{service.name}</td>
                            <td>${service.base_cost?.toFixed(2)}</td>
                            <td>
                              <span className={service.uptime_percentage < 99.9 ? 'text-warning' : 'text-success'}>
                                {service.uptime_percentage?.toFixed(2)}%
                              </span>
                            </td>
                            <td>${service.billed_cost?.toFixed(2)}</td>
                            <td>${service.credits?.toFixed(2)}</td>
                            <td>
                              <span className={`sla-badge ${service.meets_sla ? 'pass' : 'fail'}`}>
                                {service.meets_sla ? 'PASS' : 'FAIL'}
                              </span>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              ))}
            </div>
          )}

          {activeTab === 'downtime' && (
            <div className="downtime-content">
              <div className="downtime-header">
                <h3>Downtime Events</h3>
                <div className="date-controls">
                  <input
                    type="date"
                    value={dateRange.start.toISOString().split('T')[0]}
                    onChange={(e) => setDateRange({ ...dateRange, start: new Date(e.target.value) })}
                    className="date-input"
                  />
                  <span className="date-separator">to</span>
                  <input
                    type="date"
                    value={dateRange.end.toISOString().split('T')[0]}
                    onChange={(e) => setDateRange({ ...dateRange, end: new Date(e.target.value) })}
                    className="date-input"
                  />
                </div>
              </div>

              <div className="downtime-list">
                {downtime.length === 0 ? (
                  <p className="no-downtime">No downtime events in the selected period</p>
                ) : (
                  downtime.map(event => (
                    <div key={event.id} className={`downtime-event ${event.status}`}>
                      <div className="event-header">
                        <div className="event-info">
                          <span className="event-type">{event.check_type}</span>
                          <span className="event-name">{event.check_name}</span>
                          {event.domain_name && <span className="event-domain">{event.domain_name}</span>}
                        </div>
                        <span className={`event-status ${event.status}`}>
                          {event.status === 'ongoing' ? 'Ongoing' : 'Resolved'}
                        </span>
                      </div>
                      <div className="event-details">
                        <div className="event-time">
                          <span className="time-label">Started:</span>
                          <span>{new Date(event.start_time).toLocaleString()}</span>
                        </div>
                        {event.end_time && (
                          <div className="event-time">
                            <span className="time-label">Ended:</span>
                            <span>{new Date(event.end_time).toLocaleString()}</span>
                          </div>
                        )}
                        <div className="event-duration">
                          <span className="time-label">Duration:</span>
                          <span>{event.duration}</span>
                        </div>
                      </div>
                      {event.error && (
                        <div className="event-error">
                          <span className="error-label">Error:</span>
                          <span className="error-text">{event.error}</span>
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          {activeTab === 'usage' && stats && (
            <div className="usage-content">
              <h3>Usage Statistics</h3>
              {stats.service_breakdown && (
                <div className="usage-section">
                  <h4>Service Usage</h4>
                  <Charts data={stats.service_breakdown} type="service" />
                </div>
              )}
              
              <div className="usage-metrics">
                <div className="metric-card">
                  <h4>Performance Metrics</h4>
                  <div className="metrics-grid">
                    <div className="metric">
                      <span className="metric-label">Total Downtime Events</span>
                      <span className="metric-value">{stats.total_downtime_events || 0}</span>
                    </div>
                    <div className="metric">
                      <span className="metric-label">Total Downtime Hours</span>
                      <span className="metric-value">{stats.total_downtime_hours?.toFixed(2) || '0'}</span>
                    </div>
                    <div className="metric">
                      <span className="metric-label">Average Response Time</span>
                      <span className="metric-value">N/A</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default MemberDetail;