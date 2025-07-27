import React, { useState, useEffect } from 'react';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import DataTable from '../components/DataTable/DataTable';
import StatsCard from '../components/Cards/StatsCard';
import Charts from '../components/Charts/Charts';
import Loading from '../components/Loading/Loading';
import './DataView.css';

const DataView = () => {
  const [dateRange, setDateRange] = useState({
    start: new Date(new Date().setDate(new Date().getDate() - 30)),
    end: new Date()
  });
  const [activeTab, setActiveTab] = useState('country');
  const [data, setData] = useState(null);
  const [summary, setSummary] = useState(null);
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);

  useEffect(() => {
    loadInitialData();
  }, []);

  useEffect(() => {
    if (!initialLoading) {
      loadData();
      loadSummary();
    }
  }, [dateRange, activeTab, initialLoading]);

  const loadInitialData = async () => {
    setInitialLoading(true);
    try {
      await Promise.all([
        loadData(),
        loadSummary()
      ]);
    } finally {
      // Wait for animation to complete
      setTimeout(() => setInitialLoading(false), 1500);
    }
  };

  const loadData = async () => {
    setLoading(true);
    try {
      const params = {
        start: dateRange.start.toISOString().split('T')[0],
        end: dateRange.end.toISOString().split('T')[0]
      };
      let response;
      switch (activeTab) {
        case 'country':
          response = await ApiHelper.fetchRequestsByCountry(params);
          break;
        case 'asn':
          response = await ApiHelper.fetchRequestsByASN(params);
          break;
        case 'service':
          response = await ApiHelper.fetchRequestsByService(params);
          break;
        case 'member':
          response = await ApiHelper.fetchRequestsByMember(params);
          break;
        default:
          response = { data: [] };
      }
      setData(response.data);
    } catch (error) {
      console.error('Error loading data:', error);
    }
    setLoading(false);
  };

  const loadSummary = async () => {
    try {
      const params = {
        start: dateRange.start.toISOString().split('T')[0],
        end: dateRange.end.toISOString().split('T')[0]
      };
      const response = await ApiHelper.fetchRequestsSummary(params);
      setSummary(response.data);
    } catch (error) {
      console.error('Error loading summary:', error);
    }
  };

  const tabs = [
    { id: 'country', label: 'By Country', icon: '🌎' },
    { id: 'asn', label: 'By ASN', icon: '🌐' },
    { id: 'service', label: 'By Service', icon: '⚡' },
    { id: 'member', label: 'By Member', icon: '👥' }
  ];

  if (initialLoading) {
    return <Loading pageLevel={true} dataReady={true} />;
  }

  return (
    <div className="data-view fade-in">
      <div className="view-header">
        <h1>Data Analytics</h1>
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

      {summary && (
        <div className="stats-grid">
          <StatsCard
            title="Total DNS Requests"
            value={summary.total_requests?.toLocaleString() || '0'}
            icon="📊"
          />
          <StatsCard
            title="Unique Countries"
            value={summary.unique_countries || '0'}
            icon="🌍"
          />
          <StatsCard
            title="Active Members"
            value={summary.unique_members || '0'}
            icon="👥"
          />
          <StatsCard
            title="Services"
            value={summary.unique_domains || '0'}
            icon="⚡"
          />
        </div>
      )}

      <div className="data-tabs card">
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
          {loading ? (
            <div className="loading-container">
              <div className="loading-spinner"></div>
              <p>Loading data...</p>
            </div>
          ) : data && data.length > 0 ? (
            <>
              <Charts data={data} type={activeTab} />
              <DataTable data={data} type={activeTab} />
            </>
          ) : (
            <p className="no-data">No data available for the selected period</p>
          )}
        </div>
      </div>
    </div>
  );
};

export default DataView;