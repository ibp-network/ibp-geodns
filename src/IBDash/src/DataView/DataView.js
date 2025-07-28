import React, { useState, useEffect, useRef } from 'react';
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
  const [dataReady, setDataReady] = useState(false);
  const [aggregateView, setAggregateView] = useState(true);
  const [filters, setFilters] = useState({
    country: '',
    service: '',
    member: '',
    network: ''
  });
  const [filterOptions, setFilterOptions] = useState({
    countries: [],
    services: [],
    members: [],
    networks: []
  });
  const [servicesData, setServicesData] = useState([]); // Store full service data for domain mapping
  const [showSuggestions, setShowSuggestions] = useState({
    country: false,
    service: false,
    member: false,
    network: false
  });
  const [selectedCountries, setSelectedCountries] = useState([]);
  const [selectedServices, setSelectedServices] = useState([]);
  const [selectedMembers, setSelectedMembers] = useState([]);
  const [selectedNetworks, setSelectedNetworks] = useState([]);

  const countryRef = useRef(null);
  const serviceRef = useRef(null);
  const memberRef = useRef(null);
  const networkRef = useRef(null);

  useEffect(() => {
    loadInitialData();
  }, []);

  useEffect(() => {
    if (!initialLoading) {
      loadAllData();
    }
  }, [dateRange, activeTab, selectedCountries, selectedServices, selectedMembers, selectedNetworks, initialLoading]);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (countryRef.current && !countryRef.current.contains(event.target)) {
        setShowSuggestions(prev => ({ ...prev, country: false }));
      }
      if (serviceRef.current && !serviceRef.current.contains(event.target)) {
        setShowSuggestions(prev => ({ ...prev, service: false }));
      }
      if (memberRef.current && !memberRef.current.contains(event.target)) {
        setShowSuggestions(prev => ({ ...prev, member: false }));
      }
      if (networkRef.current && !networkRef.current.contains(event.target)) {
        setShowSuggestions(prev => ({ ...prev, network: false }));
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const loadInitialData = async () => {
    setInitialLoading(true);
    setDataReady(false);
    
    try {
      // Load all data in parallel
      await Promise.all([
        loadData(true),
        loadSummary(true),
        loadFilterOptions(true)
      ]);
      setDataReady(true);
    } catch (error) {
      console.error('Error loading initial data:', error);
      setDataReady(true);
    } finally {
      // Wait a bit for data ready state to propagate
      setInitialLoading(false)
    }
  };

  const loadAllData = async () => {
    setLoading(true);
    setDataReady(false);
    
    try {
      // Load data and summary in parallel
      await Promise.all([
        loadData(true),
        loadSummary(true)
      ]);
      
      // Load filter options separately as it's less critical
      loadFilterOptions(true);
      
      setDataReady(true);
    } catch (error) {
      console.error('Error loading data:', error);
      setDataReady(true);
    } finally {
      // Wait a bit for data ready state to propagate
      setLoading(false)
    }
  };

  const loadFilterOptions = async (skipLoadingState = false) => {
    if (!skipLoadingState) {
      setLoading(true);
      setDataReady(false);
    }
    
    try {
      const params = {
        start: dateRange.start.toISOString().split('T')[0],
        end: dateRange.end.toISOString().split('T')[0]
      };

      const [countryRes, serviceRes, memberRes, asnRes, servicesFullRes] = await Promise.all([
        ApiHelper.fetchRequestsByCountry(params),
        ApiHelper.fetchRequestsByService(params),
        ApiHelper.fetchRequestsByMember(params),
        ApiHelper.fetchRequestsByASN(params),
        ApiHelper.fetchServices() // Get full service data for domain mapping
      ]);

      const countriesMap = new Map();
      countryRes.data.forEach(item => {
        if (item.country && item.country_name) {
          countriesMap.set(item.country, {
            code: item.country,
            name: item.country_name
          });
        }
      });

      // Store full services data for domain mapping
      const fullServices = servicesFullRes.data?.services || [];
      setServicesData(fullServices);

      // Create service options from the full service data
      const serviceOptions = fullServices
        .filter(service => service.active)
        .map(service => service.display_name || service.name)
        .sort();

      const members = [...new Set(memberRes.data.map(item => item.member))]
        .filter(m => m && m !== '(none)')
        .sort();

      const networksMap = new Map();
      asnRes.data.forEach(item => {
        if (item.asn && item.network && item.asn !== 'Unknown') {
          networksMap.set(item.asn, {
            asn: item.asn,
            name: item.network
          });
        }
      });

      setFilterOptions({
        countries: Array.from(countriesMap.values()),
        services: serviceOptions,
        members,
        networks: Array.from(networksMap.values())
      });
    } catch (error) {
      console.error('Error loading filter options:', error);
    } finally {
      if (!skipLoadingState) {
        setDataReady(true);
        setLoading(false);
      }
    }
  };

  // Helper function to extract domains from service
  const extractDomainsFromService = (serviceName) => {
    const service = servicesData.find(s => 
      (s.display_name === serviceName) || (s.name === serviceName)
    );
    
    if (!service || !service.providers) {
      return [];
    }

    const domains = new Set();
    
    service.providers.forEach(provider => {
      provider.rpc_urls.forEach(url => {
        // Extract domain from URL
        let domain = url;
        
        // Remove protocol
        domain = domain.replace(/^(https?|wss?):\/\//, '');
        
        // Remove port
        domain = domain.replace(/:\d+.*$/, '');
        
        // Remove path
        domain = domain.replace(/\/.*$/, '');
        
        // Convert to lowercase
        domain = domain.toLowerCase();
        
        // Skip system domains
        const systemDomains = ['rpc.dotters.network', 'sys.dotters.network', 'rpc.ibp.network', 'sys.ibp.network'];
        if (!systemDomains.includes(domain)) {
          domains.add(domain);
        }
      });
    });

    return Array.from(domains);
  };

  const loadData = async (skipLoadingState = false) => {
    if (!skipLoadingState) {
      setLoading(true);
      setDataReady(false);
    }
    
    try {
      const params = {
        start: dateRange.start.toISOString().split('T')[0],
        end: dateRange.end.toISOString().split('T')[0]
      };

      // Add comma-separated filter values to params
      if (selectedCountries.length > 0) {
        params.country = selectedCountries.join(',');
      }

      // Convert selected services to domains
      if (selectedServices.length > 0) {
        const allDomains = [];
        selectedServices.forEach(serviceName => {
          const domains = extractDomainsFromService(serviceName);
          allDomains.push(...domains);
        });
        
        if (allDomains.length > 0) {
          params.domain = allDomains.join(',');
        }
      }

      if (selectedMembers.length > 0) {
        params.member = selectedMembers.join(',');
      }
      if (selectedNetworks.length > 0) {
        params.asn = selectedNetworks.map(n => n.asn).join(',');
      }

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
    } finally {
      if (!skipLoadingState) {
        setDataReady(true);
        setLoading(false);
      }
    }
  };

  const loadSummary = async (skipLoadingState = false) => {
    if (!skipLoadingState) {
      setLoading(true);
      setDataReady(false);
    }
    
    try {
      const params = {
        start: dateRange.start.toISOString().split('T')[0],
        end: dateRange.end.toISOString().split('T')[0]
      };
      const response = await ApiHelper.fetchRequestsSummary(params);
      setSummary(response.data);
    } catch (error) {
      console.error('Error loading summary:', error);
    } finally {
      if (!skipLoadingState) {
        setDataReady(true);
        setLoading(false)
      }
    }
  };

  const getFilteredSuggestions = (input, options, type) => {
    const searchTerm = input.toLowerCase();
    if (type === 'country') {
      return options.filter(country => 
        country.name.toLowerCase().includes(searchTerm) ||
        country.code.toLowerCase().includes(searchTerm)
      );
    } else if (type === 'network') {
      return options.filter(network =>
        network.name.toLowerCase().includes(searchTerm) ||
        network.asn.toLowerCase().includes(searchTerm)
      );
    } else {
      return options.filter(option =>
        option.toLowerCase().includes(searchTerm)
      );
    }
  };

  const handleTagRemove = (value, type) => {
    switch (type) {
      case 'country':
        setSelectedCountries(prev => prev.filter(c => c !== value));
        break;
      case 'service':
        setSelectedServices(prev => prev.filter(s => s !== value));
        break;
      case 'member':
        setSelectedMembers(prev => prev.filter(m => m !== value));
        break;
      case 'network':
        setSelectedNetworks(prev => prev.filter(n => n.asn !== value.asn));
        break;
    }
  };

  const handleSuggestionClick = (suggestion, type) => {
    switch (type) {
      case 'country':
        if (!selectedCountries.includes(suggestion.code)) {
          setSelectedCountries(prev => [...prev, suggestion.code]);
        }
        setFilters(prev => ({ ...prev, country: '' }));
        break;
      case 'service':
        if (!selectedServices.includes(suggestion)) {
          setSelectedServices(prev => [...prev, suggestion]);
        }
        setFilters(prev => ({ ...prev, service: '' }));
        break;
      case 'member':
        if (!selectedMembers.includes(suggestion)) {
          setSelectedMembers(prev => [...prev, suggestion]);
        }
        setFilters(prev => ({ ...prev, member: '' }));
        break;
      case 'network':
        if (!selectedNetworks.find(n => n.asn === suggestion.asn)) {
          setSelectedNetworks(prev => [...prev, suggestion]);
        }
        setFilters(prev => ({ ...prev, network: '' }));
        break;
    }
    setShowSuggestions(prev => ({ ...prev, [type]: false }));
  };

  const handleKeyDown = (e, type) => {
    if (e.key === 'Enter' && filters[type].trim()) {
      const trimmed = filters[type].trim();
      switch (type) {
        case 'country':
          const country = filterOptions.countries.find(c => 
            c.code.toLowerCase() === trimmed.toLowerCase() ||
            c.name.toLowerCase() === trimmed.toLowerCase()
          );
          if (country && !selectedCountries.includes(country.code)) {
            setSelectedCountries(prev => [...prev, country.code]);
          }
          break;
        case 'service':
          if (!selectedServices.includes(trimmed)) {
            setSelectedServices(prev => [...prev, trimmed]);
          }
          break;
        case 'member':
          if (!selectedMembers.includes(trimmed)) {
            setSelectedMembers(prev => [...prev, trimmed]);
          }
          break;
        case 'network':
          const network = filterOptions.networks.find(n =>
            n.name.toLowerCase() === trimmed.toLowerCase()
          );
          if (network && !selectedNetworks.find(n => n.asn === network.asn)) {
            setSelectedNetworks(prev => [...prev, network]);
          }
          break;
      }
      setFilters(prev => ({ ...prev, [type]: '' }));
      setShowSuggestions(prev => ({ ...prev, [type]: false }));
    }
  };

  const clearAllFilters = () => {
    setSelectedCountries([]);
    setSelectedServices([]);
    setSelectedMembers([]);
    setSelectedNetworks([]);
    setFilters({
      country: '',
      service: '',
      member: '',
      network: ''
    });
  };

  const tabs = [
    { id: 'country', label: 'By Country', icon: '🌍' },
    { id: 'asn', label: 'By ASN', icon: '🌐' },
    { id: 'service', label: 'By Service', icon: '⚡' },
    { id: 'member', label: 'By Member', icon: '👥' }
  ];

  // Show full-screen loading for initial load or when loading data
  if (initialLoading || loading) {
    return <Loading pageLevel={true} dataReady={dataReady} minDuration={500} />;
  }

  return (
    <div className="data-view fade-in">
      <div className="view-header">
        <h1>Data Analytics</h1>
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

      <div className="unified-controls-bar">
        <div className="controls-top-row">
          <div className="control-section">
            <span className="control-label">View Mode:</span>
            <div className="aggregate-checkbox">
              <input
                type="checkbox"
                id="aggregate-view"
                checked={aggregateView}
                onChange={(e) => setAggregateView(e.target.checked)}
              />
              <label htmlFor="aggregate-view">Aggregate View</label>
            </div>
          </div>
          
          <div className="control-section">
            <span className="control-label">Date Range:</span>
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

          <button
            className="clear-filters-btn"
            onClick={clearAllFilters}
            disabled={selectedCountries.length === 0 && selectedServices.length === 0 &&
                      selectedMembers.length === 0 && selectedNetworks.length === 0}
          >
            Clear All Filters
          </button>
        </div>

        <div className="filters-row">
          <div className="filter-section" ref={countryRef}>
            <span className="filter-label">Filter Countries:</span>
            <div className="filter-input-wrapper">
              <div className="selected-tags">
                {selectedCountries.map(code => (
                  <span key={code} className="filter-tag">
                    {code}
                    <button onClick={() => handleTagRemove(code, 'country')}>×</button>
                  </span>
                ))}
                <input
                  type="text"
                  placeholder="Type country name or code..."
                  value={filters.country}
                  onChange={(e) => {
                    setFilters({ ...filters, country: e.target.value });
                    setShowSuggestions({ ...showSuggestions, country: true });
                  }}
                  onFocus={() => setShowSuggestions({ ...showSuggestions, country: true })}
                  onKeyDown={(e) => handleKeyDown(e, 'country')}
                  className="filter-input"
                />
              </div>
              {showSuggestions.country && filters.country && (
                <div className="suggestions-dropdown">
                  {getFilteredSuggestions(filters.country, filterOptions.countries, 'country').map(country => (
                    <div
                      key={country.code}
                      className="suggestion-item"
                      onClick={() => handleSuggestionClick(country, 'country')}
                    >
                      <span className="suggestion-code">{country.code}</span>
                      <span className="suggestion-name">{country.name}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
          
          <div className="filter-section" ref={serviceRef}>
            <span className="filter-label">Filter Services:</span>
            <div className="filter-input-wrapper">
              <div className="selected-tags">
                {selectedServices.map(service => (
                  <span key={service} className="filter-tag">
                    {service}
                    <button onClick={() => handleTagRemove(service, 'service')}>×</button>
                  </span>
                ))}
                <input
                  type="text"
                  placeholder="Type service name..."
                  value={filters.service}
                  onChange={(e) => {
                    setFilters({ ...filters, service: e.target.value });
                    setShowSuggestions({ ...showSuggestions, service: true });
                  }}
                  onFocus={() => setShowSuggestions({ ...showSuggestions, service: true })}
                  onKeyDown={(e) => handleKeyDown(e, 'service')}
                  className="filter-input"
                />
              </div>
              {showSuggestions.service && filters.service && (
                <div className="suggestions-dropdown">
                  {getFilteredSuggestions(filters.service, filterOptions.services, 'service').map(service => (
                    <div
                      key={service}
                      className="suggestion-item"
                      onClick={() => handleSuggestionClick(service, 'service')}
                    >
                      {service}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
          
          <div className="filter-section" ref={memberRef}>
            <span className="filter-label">Filter Members:</span>
            <div className="filter-input-wrapper">
              <div className="selected-tags">
                {selectedMembers.map(member => (
                  <span key={member} className="filter-tag">
                    {member}
                    <button onClick={() => handleTagRemove(member, 'member')}>×</button>
                  </span>
                ))}
                <input
                  type="text"
                  placeholder="Type member name..."
                  value={filters.member}
                  onChange={(e) => {
                    setFilters({ ...filters, member: e.target.value });
                    setShowSuggestions({ ...showSuggestions, member: true });
                  }}
                  onFocus={() => setShowSuggestions({ ...showSuggestions, member: true })}
                  onKeyDown={(e) => handleKeyDown(e, 'member')}
                  className="filter-input"
                />
              </div>
              {showSuggestions.member && filters.member && (
                <div className="suggestions-dropdown">
                  {getFilteredSuggestions(filters.member, filterOptions.members, 'member').map(member => (
                    <div
                      key={member}
                      className="suggestion-item"
                      onClick={() => handleSuggestionClick(member, 'member')}
                    >
                      {member}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
          
          <div className="filter-section" ref={networkRef}>
            <span className="filter-label">Filter Networks:</span>
            <div className="filter-input-wrapper">
              <div className="selected-tags">
                {selectedNetworks.map(network => (
                  <span key={network.asn} className="filter-tag">
                    {network.name}
                    <button onClick={() => handleTagRemove(network, 'network')}>×</button>
                  </span>
                ))}
                <input
                  type="text"
                  placeholder="Type network name..."
                  value={filters.network}
                  onChange={(e) => {
                    setFilters({ ...filters, network: e.target.value });
                    setShowSuggestions({ ...showSuggestions, network: true });
                  }}
                  onFocus={() => setShowSuggestions({ ...showSuggestions, network: true })}
                  onKeyDown={(e) => handleKeyDown(e, 'network')}
                  className="filter-input"
                />
              </div>
              {showSuggestions.network && filters.network && (
                <div className="suggestions-dropdown">
                  {getFilteredSuggestions(filters.network, filterOptions.networks, 'network').map(network => (
                    <div
                      key={network.asn}
                      className="suggestion-item"
                      onClick={() => handleSuggestionClick(network, 'network')}
                    >
                      <span className="suggestion-name">{network.name}</span>
                      <span className="suggestion-code">{network.asn}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

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
          {data && data.length > 0 ? (
            <>
              <Charts data={data} type={activeTab} />
              <DataTable data={data} type={activeTab} aggregateView={aggregateView} />
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