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
  const [monthlyUptime, setMonthlyUptime] = useState([]);
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
      setDowntime(Array.isArray(downtimeRes.data) ? downtimeRes.data : []);
      
      // Calculate monthly uptime for the past 12 months
      calculateMonthlyUptime(memberName);
      
      setLoading(false);
    } catch (error) {
      console.error('Error loading member data:', error);
      setDowntime([]);
      setLoading(false);
    }
  };

  const calculateMonthlyUptime = async (memberName) => {
    const months = [];
    const today = new Date();
    
    for (let i = 11; i >= 0; i--) {
      const monthStart = new Date(today.getFullYear(), today.getMonth() - i, 1);
      const monthEnd = new Date(today.getFullYear(), today.getMonth() - i + 1, 0);
      
      try {
        const params = {
          start: monthStart.toISOString().split('T')[0],
          end: monthEnd.toISOString().split('T')[0]
        };
        
        const downtimeRes = await ApiHelper.fetchDowntimeEvents({ 
          member: memberName, 
          check_type: 'site',
          ...params 
        });
        
        const siteDowntime = Array.isArray(downtimeRes.data) ? downtimeRes.data : [];
        
        // Calculate total downtime hours for the month (site checks only)
        let totalDowntimeHours = 0;
        siteDowntime.forEach(event => {
          const start = new Date(event.start_time);
          const end = event.end_time ? new Date(event.end_time) : new Date();
          
          // Ensure we only count time within the month
          const eventStart = start < monthStart ? monthStart : start;
          const eventEnd = end > monthEnd ? monthEnd : end;
          
          if (eventEnd > eventStart) {
            totalDowntimeHours += (eventEnd - eventStart) / (1000 * 60 * 60);
          }
        });
        
        const totalHoursInMonth = (monthEnd - monthStart) / (1000 * 60 * 60);
        const uptime = ((totalHoursInMonth - totalDowntimeHours) / totalHoursInMonth) * 100;
        
        months.push({
          month: monthStart.toLocaleDateString('en-US', { month: 'short' }),
          year: monthStart.getFullYear(),
          uptime: Math.max(0, Math.min(100, uptime)),
          downtime: totalDowntimeHours
        });
      } catch (error) {
        console.error('Error calculating monthly uptime:', error);
        months.push({
          month: monthStart.toLocaleDateString('en-US', { month: 'short' }),
          year: monthStart.getFullYear(),
          uptime: 100,
          downtime: 0
        });
      }
    }
    
    setMonthlyUptime(months);
  };

  const getServiceStatus = (serviceName) => {
    // Check for service-specific downtime
    const serviceDowntime = downtime.filter(dt => 
      (dt.domain_name && dt.domain_name.includes(serviceName.toLowerCase())) ||
      (dt.endpoint && dt.endpoint.includes(serviceName.toLowerCase()))
    );
    
    // Check for site-level downtime
    const siteDowntime = downtime.filter(dt => dt.check_type === 'site' && !dt.end_time);
    
    if (siteDowntime.length > 0) return 'offline';
    if (serviceDowntime.some(dt => !dt.end_time)) return 'offline';
    if (serviceDowntime.length > 0) return 'degraded';
    return 'operational';
  };

  const getServiceUptime = (serviceName) => {
    if (!billing || !billing.members || billing.members.length === 0) return 100;
    
    const memberBilling = billing.members.find(m => m.name === memberName);
    if (!memberBilling || !memberBilling.services) return 100;
    
    const service = memberBilling.services.find(s => s.name === serviceName);
    return service ? service.uptime_percentage : 100;
  };

  const groupDowntimeEvents = () => {
    const grouped = {
      site: [],
      services: {}
    };

    downtime.forEach(event => {
      if (event.check_type === 'site') {
        grouped.site.push(event);
      } else {
        // Map to service name
        const serviceName = getServiceFromEvent(event);
        if (!grouped.services[serviceName]) {
          grouped.services[serviceName] = [];
        }
        grouped.services[serviceName].push(event);
      }
    });

    return grouped;
  };

  const getServiceFromEvent = (event) => {
    if (!member || !member.services) return 'Unknown Service';
    
    // Try to match domain or endpoint to a service
    for (const service of member.services) {
      if ((event.domain_name && event.domain_name.includes(service.toLowerCase())) ||
          (event.endpoint && event.endpoint.includes(service.toLowerCase()))) {
        return service;
      }
    }
    
    return event.domain_name || event.endpoint || 'Unknown Service';
  };

  const getUptimeClass = (uptime) => {
    if (uptime >= 99.99) return 'excellent';
    if (uptime >= 99.9) return 'good';
    if (uptime >= 99) return 'fair';
    return 'poor';
  };

  const getCountryFlag = (countryCode) => {
    // Simple mapping of country codes to flag emojis
    const flags = {
      'US': '🇺🇸', 'GB': '🇬🇧', 'DE': '🇩🇪', 'FR': '🇫🇷', 'JP': '🇯🇵',
      'CN': '🇨🇳', 'IN': '🇮🇳', 'BR': '🇧🇷', 'CA': '🇨🇦', 'AU': '🇦🇺',
      'NL': '🇳🇱', 'SG': '🇸🇬', 'KR': '🇰🇷', 'ES': '🇪🇸', 'IT': '🇮🇹'
    };
    return flags[countryCode] || '🌍';
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

  const groupedDowntime = groupDowntimeEvents();

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
          <div className="stat-icon">✅</div>
          <div className="stat-content">
            <div className="stat-value">{stats?.uptime_percentage?.toFixed(2) || '100'}%</div>
            <div className="stat-label">Site Uptime</div>
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
                <h3>
                  <span className="section-icon">ℹ️</span>
                  Member Information
                </h3>
                <div className="info-grid">
                  <div className="info-item">
                    <span className="info-icon">🌐</span>
                    <div className="info-content">
                      <span className="info-label">Website</span>
                      <a href={member.website} target="_blank" rel="noopener noreferrer" className="info-value link">
                        {member.website}
                      </a>
                    </div>
                  </div>
                  <div className="info-item">
                    <span className="info-icon">🔢</span>
                    <div className="info-content">
                      <span className="info-label">IPv4 Address</span>
                      <span className="info-value">{member.service_ipv4 || 'Not configured'}</span>
                    </div>
                  </div>
                  <div className="info-item">
                    <span className="info-icon">🔢</span>
                    <div className="info-content">
                      <span className="info-label">IPv6 Address</span>
                      <span className="info-value">{member.service_ipv6 || 'Not configured'}</span>
                    </div>
                  </div>
                  <div className="info-item">
                    <span className="info-icon">📍</span>
                    <div className="info-content">
                      <span className="info-label">Location</span>
                      <span className="info-value">{member.latitude?.toFixed(4)}, {member.longitude?.toFixed(4)}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="info-section">
                <h3>
                  <span className="section-icon">⚡</span>
                  Active Services
                </h3>
                <table className="services-table">
                  <thead>
                    <tr>
                      <th>Service Name</th>
                      <th>Status</th>
                      <th>Uptime</th>
                    </tr>
                  </thead>
                  <tbody>
                    {member.services?.map(service => {
                      const status = getServiceStatus(service);
                      const uptime = getServiceUptime(service);
                      return (
                        <tr key={service}>
                          <td>{service}</td>
                          <td>
                            <div className={`service-status ${status}`}>
                              <span className="service-status-icon">
                                {status === 'operational' ? '✅' : status === 'degraded' ? '⚠️' : '❌'}
                              </span>
                              <span>{status.charAt(0).toUpperCase() + status.slice(1)}</span>
                            </div>
                          </td>
                          <td>
                            <span className={`service-uptime ${uptime >= 99.9 ? 'good' : uptime >= 99 ? 'warning' : 'bad'}`}>
                              {uptime.toFixed(2)}%
                            </span>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>

              <div className="info-section">
                <h3>
                  <span className="section-icon">📅</span>
                  Monthly Uptime Calendar
                </h3>
                <div className="uptime-calendar">
                  {monthlyUptime.map((month, index) => (
                    <div key={index} className="month-item">
                      <div className="month-name">{month.month}</div>
                      <div className={`month-uptime ${getUptimeClass(month.uptime)}`}>
                        {month.uptime.toFixed(2)}%
                      </div>
                      <div className="month-status">
                        {month.downtime > 0 ? `${month.downtime.toFixed(1)}h down` : 'No downtime'}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {stats?.top_countries && stats.top_countries.length > 0 && (
                <div className="info-section">
                  <h3>
                    <span className="section-icon">🌍</span>
                    Top Countries by Requests
                  </h3>
                  <div className="countries-grid">
                    {stats.top_countries.map(country => (
                      <div key={country.country} className="country-card">
                        <div className="country-info">
                          <span className="country-flag">{getCountryFlag(country.country)}</span>
                          <span className="country-name">{country.name}</span>
                        </div>
                        <span className="country-requests">{country.requests.toLocaleString()}</span>
                      </div>
                    ))}
                  </div>
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

                  {memberBilling.services && memberBilling.services.length > 0 && (
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
                          {memberBilling.services.map(service => (
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
                  )}
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

              <div className="downtime-grouped">
                {/* Site-level downtime */}
                {groupedDowntime.site.length > 0 && (
                  <div className="downtime-group">
                    <div className="downtime-group-header">
                      <span className="downtime-group-icon">🌐</span>
                      <span className="downtime-group-title">Site-Level Issues (Affects All Services)</span>
                      <span className="downtime-group-count">{groupedDowntime.site.length}</span>
                    </div>
                    {groupedDowntime.site.map((event, index) => (
                      <div key={event.id || index} className={`downtime-event ${event.status}`}>
                        <div className="event-header">
                          <div className="event-info">
                            <span className="event-icon">🔴</span>
                            <span className="event-type">{event.check_type}</span>
                            <span className="event-name">{event.check_name}</span>
                          </div>
                          <span className={`event-status ${event.status}`}>
                            {event.status === 'ongoing' ? 'Ongoing' : 'Resolved'}
                          </span>
                        </div>
                        <div className="event-details">
                          <div className="event-time">
                            <span className="time-icon">🕐</span>
                            <span className="time-label">Started:</span>
                            <span>{new Date(event.start_time).toLocaleString()}</span>
                          </div>
                          {event.end_time && (
                            <div className="event-time">
                              <span className="time-icon">🕑</span>
                              <span className="time-label">Ended:</span>
                              <span>{new Date(event.end_time).toLocaleString()}</span>
                            </div>
                          )}
                          <div className="event-duration">
                            <span className="time-icon">⏱️</span>
                            <span className="time-label">Duration:</span>
                            <span>{event.duration}</span>
                          </div>
                        </div>
                        {event.error && (
                          <div className="event-error">
                            <span className="error-icon">⚠️</span>
                            <span className="error-label">Error:</span>
                            <span className="error-text">{event.error}</span>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}

                {/* Service-specific downtime */}
                {Object.keys(groupedDowntime.services).length > 0 && (
                  <div className="downtime-group">
                    <div className="downtime-group-header">
                      <span className="downtime-group-icon">⚡</span>
                      <span className="downtime-group-title">Service-Specific Issues</span>
                      <span className="downtime-group-count">
                        {Object.values(groupedDowntime.services).reduce((sum, events) => sum + events.length, 0)}
                      </span>
                    </div>
                    {Object.entries(groupedDowntime.services).map(([serviceName, events]) => (
                      <div key={serviceName} className="downtime-service-group">
                        <div className="downtime-service-name">{serviceName}</div>
                        {events.map((event, index) => (
                          <div key={event.id || index} className={`downtime-event ${event.status}`}>
                            <div className="event-header">
                              <div className="event-info">
                                <span className="event-icon">⚠️</span>
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
                                <span className="time-icon">🕐</span>
                                <span className="time-label">Started:</span>
                                <span>{new Date(event.start_time).toLocaleString()}</span>
                              </div>
                              {event.end_time && (
                                <div className="event-time">
                                  <span className="time-icon">🕑</span>
                                  <span className="time-label">Ended:</span>
                                  <span>{new Date(event.end_time).toLocaleString()}</span>
                                </div>
                              )}
                              <div className="event-duration">
                                <span className="time-icon">⏱️</span>
                                <span className="time-label">Duration:</span>
                                <span>{event.duration}</span>
                              </div>
                            </div>
                            {event.error && (
                              <div className="event-error">
                                <span className="error-icon">⚠️</span>
                                <span className="error-label">Error:</span>
                                <span className="error-text">{event.error}</span>
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    ))}
                  </div>
                )}

                {downtime.length === 0 && (
                  <div className="no-downtime">
                    <div className="no-downtime-icon">✅</div>
                    <p>No downtime events in the selected period</p>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'usage' && stats && (
            <div className="usage-content">
              <h3>Usage Statistics</h3>
              
              {stats.top_countries && stats.top_countries.length > 0 && (
                <div className="usage-section">
                  <h4>Country Distribution</h4>
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
              
              {stats.service_breakdown && stats.service_breakdown.length > 0 && (
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