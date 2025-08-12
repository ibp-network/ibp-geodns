import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Loading from '../components/Loading/Loading';
import { getServiceTypeIcon, getServiceTypeLabel, getNetworkTypeIcon } from '../utils/serviceUtils';
import './ServiceView.css';

const ServiceView = () => {
  const navigate = useNavigate();
  const [hierarchy, setHierarchy] = useState(null);
  const [selectedService, setSelectedService] = useState(null);
  const [members, setMembers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('relay'); // 'relay', 'system', or 'community'
  const [selectedRelayFilter, setSelectedRelayFilter] = useState('all'); // For filtering system/community by relay
  const [searchTerm, setSearchTerm] = useState('');
  const [copiedEndpoint, setCopiedEndpoint] = useState(null);

  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    try {
      const [hierarchyRes, membersRes] = await Promise.all([
        ApiHelper.fetchServicesHierarchy(),
        ApiHelper.fetchMembers()
      ]);
      
      setHierarchy(hierarchyRes.data || { relay_chains: [], orphans: [] });
      setMembers(membersRes.data || []);
      
      setLoading(false);
    } catch (error) {
      console.error('Error loading initial data:', error);
      setLoading(false);
    }
  };

  const copyToClipboard = (text, endpoint) => {
    navigator.clipboard.writeText(text);
    setCopiedEndpoint(endpoint);
    setTimeout(() => setCopiedEndpoint(null), 2000);
  };

  const getServiceMembers = (serviceName) => {
    return members.filter(member => 
      member.services?.includes(serviceName)
    );
  };

  const generateUsageExample = (service) => {
    if (service.service_type === 'RPC') {
      return {
        title: 'WebSocket RPC Connection',
        description: 'Connect to this service using any Polkadot/Substrate compatible client:',
        examples: [
          {
            label: 'Polkadot.js Example:',
            code: `import { ApiPromise, WsProvider } from '@polkadot/api';

const provider = new WsProvider('wss://${service.providers[0]?.rpc_urls[0]?.replace('wss://', '') || 'example.com'}');
const api = await ApiPromise.create({ provider });

// Query chain info
const chain = await api.rpc.system.chain();
console.log('Connected to:', chain.toString());`
          }
        ]
      };
    }
    return null;
  };

  // Get all services based on active tab
  const getFilteredServices = () => {
    if (!hierarchy) return [];
    
    let services = [];
    
    if (activeTab === 'relay') {
      // Show only relay chains
      services = hierarchy.relay_chains.map(rc => rc.relay);
    } else if (activeTab === 'system') {
      // Show system chains, optionally filtered by relay
      hierarchy.relay_chains.forEach(rc => {
        if (selectedRelayFilter === 'all' || rc.relay.name === selectedRelayFilter) {
          services = services.concat(rc.system_chains || []);
        }
      });
    } else if (activeTab === 'community') {
      // Show community chains, optionally filtered by relay
      hierarchy.relay_chains.forEach(rc => {
        if (selectedRelayFilter === 'all' || rc.relay.name === selectedRelayFilter) {
          services = services.concat(rc.community_chains || []);
        }
      });
    }

    // Apply search filter
    if (searchTerm) {
      services = services.filter(service => 
        service.display_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        service.name?.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    return services;
  };

  // Get relay chain names for filter dropdown
  const getRelayChains = () => {
    if (!hierarchy) return [];
    return hierarchy.relay_chains.map(rc => ({
      name: rc.relay.name,
      display_name: rc.relay.display_name
    }));
  };

  // Count services per category
  const getServiceCounts = () => {
    if (!hierarchy) return { relay: 0, system: 0, community: 0 };
    
    let counts = {
      relay: hierarchy.relay_chains.length,
      system: 0,
      community: 0
    };

    hierarchy.relay_chains.forEach(rc => {
      counts.system += (rc.system_chains?.length || 0);
      counts.community += (rc.community_chains?.length || 0);
    });

    return counts;
  };

  const renderServiceCard = (service) => {
    const isSelected = selectedService?.name === service.name;
    
    return (
      <div
        key={service.name}
        className={`service-card-inline ${isSelected ? 'selected' : ''}`}
        onClick={() => setSelectedService(service)}
      >
        <div className="service-card-header">
          {service.logo_url ? (
            <img 
              src={service.logo_url} 
              alt={service.display_name}
              className="service-card-logo"
              onError={(e) => {
                e.target.style.display = 'none';
                e.target.nextSibling.style.display = 'flex';
              }}
            />
          ) : null}
          <div 
            className="service-card-logo-placeholder"
            style={{ display: service.logo_url ? 'none' : 'flex' }}
          >
            {service.display_name?.substring(0, 2).toUpperCase()}
          </div>
          <div className="service-card-info">
            <div className="service-card-name">{service.display_name || service.name}</div>
            <div className="service-card-meta">
              <span className={`network-type-badge ${service.network_type?.toLowerCase()}`}>
                {getNetworkTypeIcon(service.network_type)}
                {service.network_type}
              </span>
              <span className={`status-indicator ${service.active ? 'active' : 'inactive'}`}>
                <span className={`status-dot ${service.active ? 'active' : 'inactive'}`}></span>
                {service.active ? 'Active' : 'Inactive'}
              </span>
            </div>
          </div>
        </div>
      </div>
    );
  };

  if (loading) {
    return <Loading pageLevel={true} dataReady={true} />;
  }

  const counts = getServiceCounts();
  const filteredServices = getFilteredServices();
  const shouldHighlightSearch = filteredServices.length > 8;
  // Determine if we need to apply height limitation
  const shouldLimitHeight = filteredServices.length > 10;

  return (
    <div className="service-view fade-in">
      <div className="service-header">
        <h1>Service Catalog</h1>
      </div>

      {/* Enhanced Navigation Container with Service Cards Inside */}
      <div className="service-nav-container">
        {/* Service Tabs */}
        <div className="service-tabs">
          <button
            className={`service-tab ${activeTab === 'relay' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('relay');
              setSelectedRelayFilter('all');
              setSearchTerm('');
            }}
          >
            <span className="tab-icon">🗿</span>
            <span className="tab-label">Relay Chains</span>
            <span className="tab-count">{counts.relay}</span>
          </button>
          <button
            className={`service-tab ${activeTab === 'system' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('system');
              setSearchTerm('');
            }}
          >
            <span className="tab-icon">🏛️</span>
            <span className="tab-label">System Chains</span>
            <span className="tab-count">{counts.system}</span>
          </button>
          <button
            className={`service-tab ${activeTab === 'community' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('community');
              setSearchTerm('');
            }}
          >
            <span className="tab-icon">👥</span>
            <span className="tab-label">Community Chains</span>
            <span className="tab-count">{counts.community}</span>
          </button>
        </div>

        {/* Filters Bar */}
        <div className="service-filters">
          {/* Search - highlighted when needed */}
          <div className={`search-container ${shouldHighlightSearch ? 'highlighted' : ''}`}>
            <span className="search-icon">🔍</span>
            <input
              type="text"
              placeholder={shouldHighlightSearch ? "Search to filter results..." : "Search services..."}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="service-search-input"
            />
            {shouldHighlightSearch && (
              <span className="search-hint">Too many results - use search to filter</span>
            )}
          </div>

          {/* Relay Filter (only for system/community tabs) */}
          {(activeTab === 'system' || activeTab === 'community') && (
            <select
              className="relay-filter"
              value={selectedRelayFilter}
              onChange={(e) => setSelectedRelayFilter(e.target.value)}
            >
              <option value="all">All Relay Chains</option>
              {getRelayChains().map(relay => (
                <option key={relay.name} value={relay.name}>
                  {relay.display_name}
                </option>
              ))}
            </select>
          )}
        </div>

        {/* Services Grid Inside Navigation Container - with conditional height limitation */}
        <div className={`services-grid-inline ${shouldLimitHeight ? 'limited-height' : ''}`}>
          {filteredServices.length > 0 ? (
            filteredServices.map(service => renderServiceCard(service))
          ) : (
            <div className="no-services-inline">
              <span className="no-services-icon">🔭</span>
              <p>No services found</p>
            </div>
          )}
        </div>
      </div>

      {/* Service Detail Section - Now properly expandable */}
      {selectedService && (
        <div className="service-detail-section">
          <div className="detail-header">
            {selectedService.logo_url ? (
              <img 
                src={selectedService.logo_url} 
                alt={selectedService.display_name}
                className="detail-header-logo"
                onError={(e) => {
                  e.target.style.display = 'none';
                  e.target.nextSibling.style.display = 'flex';
                }}
              />
            ) : null}
            <div 
              className="detail-header-logo-placeholder"
              style={{ display: selectedService.logo_url ? 'none' : 'flex' }}
            >
              {selectedService.display_name?.substring(0, 2).toUpperCase()}
            </div>
            <div className="detail-header-info">
              <h2>{selectedService.display_name || selectedService.name}</h2>
              <div className="detail-subtitle">
                <div className={`service-type-badge ${selectedService.service_type?.toLowerCase()}`}>
                  {getServiceTypeIcon(selectedService.service_type)}
                  {getServiceTypeLabel(selectedService.service_type)}
                </div>
                <span>•</span>
                <span className={`network-type-badge ${selectedService.network_type?.toLowerCase()}`}>
                  {getNetworkTypeIcon(selectedService.network_type)}
                  {selectedService.network_type}
                </span>
                {selectedService.relay_network && (
                  <>
                    <span>•</span>
                    <span>On {selectedService.relay_network}</span>
                  </>
                )}
                <span>•</span>
                <div className={`status-indicator ${selectedService.active ? 'active' : 'inactive'}`}>
                  <span className={`status-dot ${selectedService.active ? 'active' : 'inactive'}`}></span>
                  {selectedService.active ? 'Active' : 'Inactive'}
                </div>
              </div>
            </div>
            <button
              className="close-detail-btn"
              onClick={() => setSelectedService(null)}
            >
              ✕
            </button>
          </div>

          <div className="detail-content-expanded">
            {/* Service Information */}
            <div className="info-section">
              <h3>
                <span>ℹ️</span>
                Service Information
              </h3>
              <div className="info-grid">
                <div className="info-item">
                  <span className="info-label">Service Name</span>
                  <span className="info-value">{selectedService.display_name || selectedService.name}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">Required Level</span>
                  <span className="info-value">Level {selectedService.level_required || 'N/A'}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">Network</span>
                  <span className="info-value">{selectedService.network_name || 'N/A'}</span>
                </div>
                {selectedService.relay_network && (
                  <div className="info-item">
                    <span className="info-label">Relay Chain</span>
                    <span className="info-value">{selectedService.relay_network}</span>
                  </div>
                )}
                {selectedService.website_url && (
                  <div className="info-item">
                    <span className="info-label">Website</span>
                    <a href={selectedService.website_url} target="_blank" rel="noopener noreferrer" className="info-value link">
                      {selectedService.website_url}
                    </a>
                  </div>
                )}
              </div>
              {selectedService.description && (
                <div style={{ marginTop: '16px' }}>
                  <span className="info-label">Description</span>
                  <p style={{ marginTop: '8px', lineHeight: '1.6' }}>{selectedService.description}</p>
                </div>
              )}
            </div>

            {/* Resource Requirements */}
            {selectedService.resources && (
              <div className="info-section">
                <h3>
                  <span>💻</span>
                  Resource Requirements (Per Node)
                </h3>
                <div className="resource-requirements">
                  <div className="resource-item">
                    <div className="resource-value">{selectedService.resources.cores}</div>
                    <div className="resource-label">CPU Cores</div>
                  </div>
                  <div className="resource-item">
                    <div className="resource-value">{selectedService.resources.memory}</div>
                    <div className="resource-label">GB RAM</div>
                  </div>
                  <div className="resource-item">
                    <div className="resource-value">{selectedService.resources.disk}</div>
                    <div className="resource-label">GB Disk</div>
                  </div>
                  <div className="resource-item">
                    <div className="resource-value">{selectedService.resources.bandwidth}</div>
                    <div className="resource-label">GB Bandwidth</div>
                  </div>
                  <div className="resource-item">
                    <div className="resource-value">{selectedService.resources.nodes}</div>
                    <div className="resource-label">Nodes</div>
                  </div>
                </div>
              </div>
            )}

            {/* Usage Instructions */}
            {generateUsageExample(selectedService) && (
              <div className="usage-section">
                <h3>
                  <span>📖</span>
                  How to Use This Service
                </h3>
                <p style={{ marginBottom: '16px' }}>
                  {generateUsageExample(selectedService).description}
                </p>
                {generateUsageExample(selectedService).examples.map((example, index) => (
                  <div key={index}>
                    <div className="code-header">
                      {example.label}
                      <button 
                        className={`copy-button ${copiedEndpoint === index ? 'copied' : ''}`}
                        onClick={() => copyToClipboard(example.code, index)}
                      >
                        {copiedEndpoint === index ? 'Copied!' : 'Copy'}
                      </button>
                    </div>
                    <div className="code-block">
                      <pre style={{ margin: 0 }}>{example.code}</pre>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Endpoints */}
            {selectedService.providers && selectedService.providers.length > 0 && (
              <div className="info-section">
                <h3>
                  <span>🌐</span>
                  Service Endpoints
                </h3>
                <div className="endpoints-list">
                  {selectedService.providers.map((provider, index) => (
                    <div key={index} className="endpoint-item">
                      <div className="endpoint-header">
                        <span className="endpoint-provider">{provider.name}</span>
                        <button 
                          className={`copy-button ${copiedEndpoint === `provider-${index}` ? 'copied' : ''}`}
                          onClick={() => copyToClipboard(provider.rpc_urls.join('\n'), `provider-${index}`)}
                        >
                          {copiedEndpoint === `provider-${index}` ? 'Copied!' : 'Copy All'}
                        </button>
                      </div>
                      {provider.rpc_urls.map((url, urlIndex) => (
                        <div key={urlIndex} className="endpoint-url">{url}</div>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Members Providing This Service */}
            <div className="members-section">
              <h3>
                <span>👥</span>
                Members Providing This Service ({getServiceMembers(selectedService.name).length})
              </h3>
              <div className="members-grid">
                {getServiceMembers(selectedService.name).map(member => (
                  <div 
                    key={member.name} 
                    className="service-member-card"
                    onClick={() => navigate(`/members/${member.name}`)}
                  >
                    <div className="service-member-logo">
                      {member.logo ? (
                        <img 
                          src={member.logo} 
                          alt={member.name}
                          onError={(e) => {
                            e.target.style.display = 'none';
                            e.target.parentElement.innerHTML = `<div class="service-member-placeholder">${member.name.substring(0, 2).toUpperCase()}</div>`;
                          }}
                        />
                      ) : (
                        <div className="service-member-placeholder">
                          {member.name.substring(0, 2).toUpperCase()}
                        </div>
                      )}
                    </div>
                    
                    <div className="service-member-info">
                      <div className="service-member-name">{member.name}</div>
                      <div className="service-member-details">
                        <div className="service-member-level">
                          <span>🏆</span>
                          <span>Level {member.level}</span>
                        </div>
                        <div className="service-member-region">
                          <span>📍</span>
                          <span>{member.region}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ServiceView;