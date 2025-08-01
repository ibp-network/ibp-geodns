import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Charts from '../components/Charts/Charts';
import Loading from '../components/Loading/Loading';
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

  const calculateSiteUptime = (downtimeEvents, startDate, endDate, serviceCount = 0) => {
    if (serviceCount === 0) return 100; // No services means 100% uptime
    
    // Calculate total hours in period
    const totalHours = (endDate - startDate) / (1000 * 60 * 60);
    const totalServiceHours = totalHours * serviceCount;
    
    let totalDowntimeServiceHours = 0;
    const currentTime = new Date();
    
    // Group downtime by service
    const serviceDowntimeMap = new Map();
    
    downtimeEvents.forEach(event => {
      const eventStart = new Date(event.start_time);
      const eventEnd = event.end_time ? new Date(event.end_time) : currentTime;
      
      // Clamp to date range
      const start = eventStart < startDate ? startDate : eventStart;
      const end = eventEnd > endDate ? endDate : eventEnd;
      
      if (start >= end) return; // Skip if outside date range
      
      const downtimeHours = (end - start) / (1000 * 60 * 60);
      
      if (event.check_type === 'site') {
        // Site downtime affects ALL services
        for (let i = 0; i < serviceCount; i++) {
          const serviceKey = `service_${i}`;
          if (!serviceDowntimeMap.has(serviceKey)) {
            serviceDowntimeMap.set(serviceKey, []);
          }
          serviceDowntimeMap.get(serviceKey).push({ start, end });
        }
      } else {
        // Service-specific downtime
        const serviceName = getServiceFromEvent(event);
        if (!serviceDowntimeMap.has(serviceName)) {
          serviceDowntimeMap.set(serviceName, []);
        }
        serviceDowntimeMap.get(serviceName).push({ start, end });
      }
    });
    
    // Calculate total downtime hours with overlap handling per service
    serviceDowntimeMap.forEach((periods, serviceName) => {
      // Sort periods by start time
      const sortedPeriods = periods.sort((a, b) => a.start - b.start);
      
      // Merge overlapping periods
      const mergedPeriods = [];
      sortedPeriods.forEach(period => {
        if (mergedPeriods.length === 0 || period.start > mergedPeriods[mergedPeriods.length - 1].end) {
          mergedPeriods.push({ ...period });
        } else {
          const lastPeriod = mergedPeriods[mergedPeriods.length - 1];
          lastPeriod.end = period.end > lastPeriod.end ? period.end : lastPeriod.end;
        }
      });
      
      // Sum up the merged periods
      mergedPeriods.forEach(period => {
        totalDowntimeServiceHours += (period.end - period.start) / (1000 * 60 * 60);
      });
    });
    
    // Calculate uptime percentage
    const uptimeServiceHours = totalServiceHours - totalDowntimeServiceHours;
    const uptimePercentage = (uptimeServiceHours / totalServiceHours) * 100;
    
    return Math.max(0, Math.min(100, uptimePercentage));
  };

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
      
      // Calculate proper site uptime from downtime events
      const downtimeData = Array.isArray(downtimeRes.data) ? downtimeRes.data : [];
      setDowntime(downtimeData);
      
      // Calculate site uptime based on total service hours
      const serviceCount = memberData?.services?.length || 0;
      const siteUptime = calculateSiteUptime(downtimeData, dateRange.start, dateRange.end, serviceCount);
      
      // Update stats with calculated uptime
      const updatedStats = {
        ...statsRes.data,
        uptime_percentage: siteUptime
      };
      setStats(updatedStats);
      
      setBilling(billingRes.data);
      
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
    const currentTime = new Date();
    
    for (let i = 11; i >= 0; i--) {
      const monthStart = new Date(today.getFullYear(), today.getMonth() - i, 1);
      const monthEnd = new Date(today.getFullYear(), today.getMonth() - i + 1, 0, 23, 59, 59, 999);
      
      try {
        const params = {
          start: monthStart.toISOString().split('T')[0],
          end: monthEnd.toISOString().split('T')[0]
        };
        
        // Get all downtime for the month
        const downtimeRes = await ApiHelper.fetchDowntimeEvents({ 
          member: memberName, 
          ...params 
        });
        
        const allDowntime = Array.isArray(downtimeRes.data) ? downtimeRes.data : [];
        
        // Calculate site uptime for the month based on service hours
        const serviceCount = member?.services?.length || 0;
        const uptime = calculateSiteUptime(allDowntime, monthStart, monthEnd, serviceCount);
        
        // Calculate total downtime hours (keeping this for display)
        const totalHours = (monthEnd - monthStart) / (1000 * 60 * 60);
        const totalServiceHours = totalHours * serviceCount;
        const downtimeServiceHours = totalServiceHours * (1 - uptime / 100);
        const avgDowntimeHours = serviceCount > 0 ? downtimeServiceHours / serviceCount : 0;
        
        months.push({
          month: monthStart.toLocaleDateString('en-US', { month: 'short' }),
          year: monthStart.getFullYear(),
          uptime: uptime,
          downtime: avgDowntimeHours
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
    const serviceDowntime = downtime.filter(dt => {
      const eventService = getServiceFromEvent(dt);
      return eventService === serviceName && !dt.end_time;
    });
    
    // Check for site-level downtime
    const siteDowntime = downtime.filter(dt => dt.check_type === 'site' && !dt.end_time);
    
    if (siteDowntime.length > 0) return 'offline';
    if (serviceDowntime.length > 0) return 'offline';
    
    // Check if service had downtime but is now resolved
    const hadDowntime = downtime.some(dt => {
      const eventService = getServiceFromEvent(dt);
      return eventService === serviceName;
    });
    
    if (hadDowntime) return 'degraded';
    return 'operational';
  };

  const getServiceUptime = (serviceName, customDateRange = null) => {
    // Use custom date range if provided (for billing calculations)
    const effectiveDateRange = customDateRange || dateRange;
    
    // Calculate service-specific uptime based on downtime events
    const serviceDowntime = downtime.filter(dt => {
      const eventService = getServiceFromEvent(dt);
      return eventService === serviceName;
    });
    
    // Also include site-level downtime as it affects all services
    const siteDowntime = downtime.filter(dt => dt.check_type === 'site');
    
    const allRelevantDowntime = [...serviceDowntime, ...siteDowntime];
    
    if (allRelevantDowntime.length === 0) return 100;
    
    // Calculate total hours in period
    const totalHours = (effectiveDateRange.end - effectiveDateRange.start) / (1000 * 60 * 60);
    let totalDowntimeHours = 0;
    const currentTime = new Date();
    
    // Sort and merge overlapping periods
    const sortedEvents = allRelevantDowntime.sort((a, b) => 
      new Date(a.start_time) - new Date(b.start_time)
    );
    
    const mergedPeriods = [];
    
    sortedEvents.forEach(event => {
      const eventStart = new Date(event.start_time);
      const eventEnd = event.end_time ? new Date(event.end_time) : currentTime;
      
      const start = eventStart < effectiveDateRange.start ? effectiveDateRange.start : eventStart;
      const end = eventEnd > effectiveDateRange.end ? effectiveDateRange.end : eventEnd;
      
      if (start < end) {
        if (mergedPeriods.length === 0 || start > mergedPeriods[mergedPeriods.length - 1].end) {
          mergedPeriods.push({ start, end });
        } else {
          const lastPeriod = mergedPeriods[mergedPeriods.length - 1];
          lastPeriod.end = end > lastPeriod.end ? end : lastPeriod.end;
        }
      }
    });
    
    mergedPeriods.forEach(period => {
      totalDowntimeHours += (period.end - period.start) / (1000 * 60 * 60);
    });
    
    const uptimeHours = totalHours - totalDowntimeHours;
    const uptimePercentage = (uptimeHours / totalHours) * 100;
    
    return Math.max(0, Math.min(100, uptimePercentage));
  };

  const calculateBillingWithActualUptime = (billingData) => {
    if (!billingData || !billingData.members) return billingData;
    
    // Get the current month's date range for billing
    const now = new Date();
    const billingStartDate = new Date(now.getFullYear(), now.getMonth(), 1);
    const billingEndDate = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59, 999);
    
    const updatedBilling = {
      ...billingData,
      members: billingData.members.map(memberBilling => {
        if (memberBilling.name !== memberName) return memberBilling;
        
        const updatedServices = memberBilling.services?.map(service => {
          // Calculate actual uptime for this service based on downtime events
          const actualUptime = getServiceUptime(service.name, { start: billingStartDate, end: billingEndDate });
          
          // Determine if service meets SLA (99.9%)
          const meetsSlaNow = actualUptime >= 99.9;
          
          // Calculate credits based on actual uptime
          let credits = 0;
          if (actualUptime < 99.9 && actualUptime >= 99.0) {
            credits = service.base_cost * 0.1; // 10% credit
          } else if (actualUptime < 99.0 && actualUptime >= 95.0) {
            credits = service.base_cost * 0.25; // 25% credit
          } else if (actualUptime < 95.0) {
            credits = service.base_cost * 0.5; // 50% credit
          }
          
          const billedCost = service.base_cost - credits;
          
          return {
            ...service,
            uptime_percentage: actualUptime,
            meets_sla: meetsSlaNow,
            credits: credits,
            billed_cost: billedCost
          };
        }) || [];
        
        // Recalculate totals
        const totalBaseCost = updatedServices.reduce((sum, s) => sum + (s.base_cost || 0), 0);
        const totalCredits = updatedServices.reduce((sum, s) => sum + (s.credits || 0), 0);
        const totalBilled = totalBaseCost - totalCredits;
        
        return {
          ...memberBilling,
          services: updatedServices,
          total_base_cost: totalBaseCost,
          total_credits: totalCredits,
          total_billed: totalBilled
        };
      })
    };
    
    return updatedBilling;
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

  // Convert domain name to service name
  const domainToServiceName = (domainName) => {
    if (!domainName) return null;
    
    // Remove common suffixes
    let serviceName = domainName
      .replace('.ibp.network', '')
      .replace('.dotters.network', '');
    
    // Convert to title case with hyphens
    serviceName = serviceName.split('-').map(part => 
      part.charAt(0).toUpperCase() + part.slice(1).toLowerCase()
    ).join('-');
    
    return serviceName;
  };

  const getServiceFromEvent = (event) => {
    if (!member || !member.services) return 'Unknown Service';
    
    // Try to get service name from domain
    if (event.domain_name) {
      const serviceName = domainToServiceName(event.domain_name);
      
      // Find matching service in member's service list (case-insensitive)
      const matchingService = member.services.find(s => 
        s.toLowerCase() === serviceName.toLowerCase()
      );
      
      if (matchingService) {
        return matchingService;
      }
    }
    
    // Fallback to domain name or endpoint
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
    return <Loading pageLevel={true} dataReady={false} />;
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
  
  // Calculate billing with actual uptime including ongoing events
  const actualBilling = calculateBillingWithActualUptime(billing);

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
            <div className="stat-label">Total DNS Requests</div>
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
            <div className="stat-value">{stats?.uptime_percentage?.toFixed(2) || '100.00'}%</div>
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
                      <div className="month-name">{month.month} {month.year}</div>
                      <div className={`month-uptime ${getUptimeClass(month.uptime)}`}>
                        {month.uptime.toFixed(2)}%
                      </div>
                      <div className="month-status">
                        {month.downtime > 0 ? `${month.downtime.toFixed(1)}h avg down` : 'No downtime'}
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

          {activeTab === 'billing' && actualBilling && (
            <div className="billing-content">
              <h3>Current Month Billing</h3>
              {actualBilling.members?.map(memberBilling => (
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