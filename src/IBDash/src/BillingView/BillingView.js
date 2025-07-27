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
      setOverviewPDFs(overviews.slice(0, 3)); // Show last 3 overview PDFs
      
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
      const billingRes = await ApiHelper.fetchBillingBreakdown({ 
        member: memberName,
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

  const filteredMembers = members.filter(member =>
    member.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

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
        <div className="overview-pdfs">
          <span style={{ marginRight: '12px', color: 'var(--text-secondary)' }}>
            Recent Overviews:
          </span>
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
      </div>

      <div className="billing-content">
        <div className="members-panel">
          <div className="members-panel-header">
            <h2>Members</h2>
            <input
              type="text"
              placeholder="Search members..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="member-search"
            />
          </div>
          <div className="members-list">
            {filteredMembers.map(member => (
              <div
                key={member.name}
                className={`member-item ${selectedMember?.name === member.name ? 'active' : ''}`}
                onClick={() => setSelectedMember(member)}
              >
                <div className="member-info">
                  <div className="member-name">{member.name}</div>
                  <div className="member-stats">
                    <span>Services: {member.services?.length || 0}</span>
                    <span>Region: {member.region}</span>
                  </div>
                </div>
                <div className="member-level">Level {member.level}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="detail-panel">
          {selectedMember ? (
            <>
              <div className="detail-header">
                <h2>{selectedMember.name}</h2>
                <div className="detail-subtitle">
                  Level {selectedMember.level} • {selectedMember.region}
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
                        Current Month Billing
                      </h3>
                      
                      {memberBilling.members?.map(member => (
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
                            <div className="stat-card">
                              <div className="stat-label">SLA Credits</div>
                              <div className="stat-value success">
                                ${member.total_credits?.toFixed(2)}
                              </div>
                            </div>
                            <div className="stat-card">
                              <div className="stat-label">SLA Status</div>
                              <div className={`stat-value ${member.meets_sla ? 'success' : 'error'}`}>
                                {member.meets_sla ? 'PASS' : 'FAIL'}
                              </div>
                            </div>
                          </div>

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
              <p>Choose from the list on the left to see billing information</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default BillingView;