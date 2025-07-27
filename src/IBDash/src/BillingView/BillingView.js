import React, { useState, useEffect } from 'react';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Loading from '../components/Loading/Loading';
import './BillingView.css';

const BillingView = () => {
  const [members, setMembers] = useState([]);
  const [selectedMember, setSelectedMember] = useState(null);
  const [memberBilling, setMemberBilling] = useState(null);
  const [historicalPDFs, setHistoricalPDFs] = useState([]);
  const [overviewPDFs, setOverviewPDFs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadInitialData();
  }, []);

  useEffect(() => {
    if (selectedMember) {
      loadMemberBilling(selectedMember.name);
      loadMemberPDFs(selectedMember.name);
    }
  }, [selectedMember]);

  const loadInitialData = async () => {
    try {
      // Load members
      const membersRes = await ApiHelper.fetchMembers();
      setMembers(membersRes.data || []);
      
      // Load overview PDFs
      const pdfsRes = await ApiHelper.fetchBillingPDFs();
      const overviews = [];
      if (pdfsRes.data && pdfsRes.data.data) {
        pdfsRes.data.data.forEach(monthGroup => {
          const overview = monthGroup.pdfs.find(pdf => pdf.is_overview);
          if (overview) {
            overviews.push({
              ...overview,
              year: monthGroup.year,
              month: monthGroup.month
            });
          }
        });
      }
      setOverviewPDFs(overviews.slice(0, 6)); // Show last 6 overview PDFs
      
      setLoading(false);
    } catch (error) {
      console.error('Error loading initial data:', error);
      setLoading(false);
    }
  };

  const loadMemberBilling = async (memberName) => {
    setDetailLoading(true);
    try {
      // Get current month billing
      const now = new Date();
      const currentMonth = now.getMonth() + 1;
      const currentYear = now.getFullYear();
      
      const billingRes = await ApiHelper.fetchBillingBreakdown({ 
        member: memberName,
        month: currentMonth,
        year: currentYear,
        include_downtime: true 
      });
      setMemberBilling(billingRes.data);
    } catch (error) {
      console.error('Error loading member billing:', error);
    }
    setDetailLoading(false);
  };

  const loadMemberPDFs = async (memberName) => {
    try {
      const pdfsRes = await ApiHelper.fetchBillingPDFs({ member: memberName });
      const memberPDFs = [];
      
      if (pdfsRes.data && pdfsRes.data.data) {
        pdfsRes.data.data.forEach(monthGroup => {
          const memberPDF = monthGroup.pdfs.find(pdf => 
            !pdf.is_overview && pdf.member_name === memberName
          );
          if (memberPDF) {
            memberPDFs.push({
              ...memberPDF,
              year: monthGroup.year,
              month: monthGroup.month
            });
          }
        });
      }
      
      setHistoricalPDFs(memberPDFs.slice(0, 10)); // Show last 10 PDFs
    } catch (error) {
      console.error('Error loading member PDFs:', error);
    }
  };

  const downloadPDF = async (pdf, isOverview = false) => {
    const params = {
      year: pdf.year,
      month: pdf.month
    };
    
    if (isOverview) {
      params.type = 'overview';
    } else {
      params.member = pdf.member_name;
    }
    
    try {
      const response = await ApiHelper.downloadBillingPDF(params);
      const blob = new Blob([response.data], { type: 'application/pdf' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = pdf.file_name;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Error downloading PDF:', error);
    }
  };

  const filteredMembers = members
    .filter(member => member.name.toLowerCase().includes(searchTerm.toLowerCase()))
    .sort((a, b) => a.name.localeCompare(b.name));

  const formatMonth = (year, month) => {
    const date = new Date(year, parseInt(month) - 1);
    return date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
  };

  const getUptimeClass = (uptime) => {
    if (uptime >= 99.99) return 'good';
    if (uptime >= 99.9) return 'warning';
    return 'error';
  };

  if (loading) {
    return <Loading pageLevel={true} dataReady={true} />;
  }

  return (
    <div className="billing-view fade-in">
      <div className="billing-header">
        <h1>Billing Management</h1>
      </div>

      {/* Overview PDFs Bar */}
      <div className="overview-pdfs-bar">
        <span className="overview-label">📊 Monthly Overviews:</span>
        {overviewPDFs.map((pdf, index) => (
          <button
            key={index}
            className="overview-pdf-button"
            onClick={() => downloadPDF(pdf, true)}
          >
            📄 {formatMonth(pdf.year, pdf.month)}
          </button>
        ))}
      </div>

      {/* Members Navigation Bar */}
      <div className="members-nav-bar">
        <div className="members-nav-header">
          <h2>Select Member</h2>
          <input
            type="text"
            placeholder="Search members..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="member-search"
          />
        </div>
        <div className="members-horizontal-list">
          {filteredMembers.map(member => (
            <div
              key={member.name}
              className={`member-nav-item ${selectedMember?.name === member.name ? 'active' : ''}`}
              onClick={() => setSelectedMember(member)}
            >
              {member.logo ? (
                <img 
                  src={member.logo} 
                  alt={member.name} 
                  className="member-logo-small"
                  onError={(e) => {
                    e.target.style.display = 'none';
                    e.target.nextSibling.style.display = 'flex';
                  }}
                />
              ) : null}
              <div 
                className="member-logo-placeholder" 
                style={{ display: member.logo ? 'none' : 'flex' }}
              >
                {member.name.substring(0, 2).toUpperCase()}
              </div>
              <div className="member-nav-name">{member.name}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Detail Panel */}
      <div className="detail-panel">
        {selectedMember ? (
          <>
            <div className="detail-header">
              {selectedMember.logo ? (
                <img 
                  src={selectedMember.logo} 
                  alt={selectedMember.name} 
                  className="detail-header-logo"
                  onError={(e) => {
                    e.target.style.display = 'none';
                    e.target.nextSibling.style.display = 'flex';
                  }}
                />
              ) : null}
              <div 
                className="detail-header-logo-placeholder" 
                style={{ display: selectedMember.logo ? 'none' : 'flex' }}
              >
                {selectedMember.name.substring(0, 2).toUpperCase()}
              </div>
              <div className="detail-header-info">
                <h2>{selectedMember.name}</h2>
                <div className="detail-subtitle">
                  <span>Level {selectedMember.level}</span>
                  <span>•</span>
                  <span>{selectedMember.region}</span>
                  <span>•</span>
                  <span>{selectedMember.services?.length || 0} Active Services</span>
                </div>
              </div>
            </div>
            <div className="detail-content">
              {detailLoading ? (
                <div className="loading-container">
                  <div className="loading-spinner"></div>
                  <p>Loading billing details...</p>
                </div>
              ) : memberBilling ? (
                <>
                  {/* Current Month Section */}
                  <div className="current-month-section">
                    <h3 className="section-title">
                      <span>💵</span>
                      Current Month Billing (Month-to-Date)
                    </h3>
                    
                    {memberBilling.members?.filter(m => m.name === selectedMember.name).map(member => (
                      <div key={member.name}>
                        <div className="current-month-stats">
                          <div className="stat-card">
                            <div className="stat-label">Base Cost</div>
                            <div className="stat-value">${member.total_base_cost?.toFixed(2)}</div>
                          </div>
                          <div className="stat-card">
                            <div className="stat-label">Billed Amount</div>
                            <div className="stat-value">${member.total_billed?.toFixed(2)}</div>
                          </div>
                          <div className="stat-card success">
                            <div className="stat-label">SLA Credits</div>
                            <div className="stat-value success">
                              ${member.total_credits?.toFixed(2)}
                            </div>
                          </div>
                          <div className={`stat-card ${member.meets_sla ? 'success' : 'error'}`}>
                            <div className="stat-label">SLA Status</div>
                            <div className={`stat-value ${member.meets_sla ? 'success' : 'error'}`}>
                              {member.meets_sla ? 'PASS' : 'FAIL'}
                            </div>
                          </div>
                        </div>
                        
                        {/* Calculate and display site uptime */}
                        {member.services && member.services.length > 0 && (() => {
                          const totalUptime = member.services.reduce((sum, svc) => sum + (svc.uptime_percentage || 0), 0);
                          const avgUptime = totalUptime / member.services.length;
                          return (
                            <div style={{ marginBottom: '20px' }}>
                              <div className="stat-card">
                                <div className="stat-label">Site Uptime (Average)</div>
                                <div className={`stat-value ${avgUptime >= 99.9 ? 'success' : avgUptime >= 99 ? 'warning' : 'error'}`}>
                                  {avgUptime.toFixed(2)}%
                                </div>
                              </div>
                            </div>
                          );
                        })()}

                        {/* Service Breakdown */}
                        {member.services && member.services.length > 0 && (
                          <div className="services-breakdown">
                            <h4 className="section-title">Service Breakdown</h4>
                            <table className="service-table">
                              <thead>
                                <tr>
                                  <th>Service</th>
                                  <th>Base Cost</th>
                                  <th>Uptime</th>
                                  <th>Billed</th>
                                  <th>Credits</th>
                                  <th>SLA</th>
                                </tr>
                              </thead>
                              <tbody>
                                {member.services.map(service => (
                                  <tr key={service.name}>
                                    <td>{service.name}</td>
                                    <td>${service.base_cost?.toFixed(2)}</td>
                                    <td>
                                      <span className={`uptime-badge ${getUptimeClass(service.uptime_percentage)}`}>
                                        {service.uptime_percentage?.toFixed(2)}%
                                      </span>
                                    </td>
                                    <td>${service.billed_cost?.toFixed(2)}</td>
                                    <td>${service.credits?.toFixed(2)}</td>
                                    <td>
                                      <span className={`sla-status ${service.meets_sla ? 'pass' : 'fail'}`}>
                                        {service.meets_sla ? '✓ PASS' : '✗ FAIL'}
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

                  {/* Historical PDFs */}
                  <div className="historical-section">
                    <h3 className="section-title">
                      <span>📋</span>
                      Historical Billing PDFs
                    </h3>
                    {historicalPDFs.length > 0 ? (
                      <div className="pdf-list">
                        {historicalPDFs.map((pdf, index) => (
                          <div key={index} className="pdf-item">
                            <div className="pdf-info">
                              <div className="pdf-month">
                                {formatMonth(pdf.year, pdf.month)}
                              </div>
                              <div className="pdf-stats">
                                <span>Size: {(pdf.file_size / 1024).toFixed(1)} KB</span>
                                <span>•</span>
                                <span>Generated: {new Date(pdf.modified_time).toLocaleDateString()}</span>
                              </div>
                            </div>
                            <button
                              className="pdf-download"
                              onClick={() => downloadPDF(pdf)}
                            >
                              <span>⬇</span>
                              Download PDF
                            </button>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="no-data">
                        No historical PDFs available
                      </div>
                    )}
                  </div>
                </>
              ) : (
                <div className="no-data">
                  No billing data available
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="placeholder-content">
            <div className="placeholder-icon">💰</div>
            <h3>Select a member to view billing details</h3>
            <p>Choose from the list above to see billing information</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default BillingView;