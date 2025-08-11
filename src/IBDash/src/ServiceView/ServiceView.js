import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Loading from '../components/Loading/Loading';
import { getServiceTypeIcon, getServiceTypeLabel, getNetworkTypeIcon } from '../utils/serviceUtils';
import './ServiceView.css';

const ServiceView = () => {
  const navigate = useNavigate();
  const [hierarchy, setHierarchy] = useState(null);
  const [selectedRelay, setSelectedRelay] = useState(null);
  const [selectedService, setSelectedService] = useState(null);
  const [members, setMembers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState('hierarchy'); // 'hierarchy' or 'flat'
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
      
      // Auto-select first relay if available
      if (hierarchyRes.data?.relay_chains?.length > 0) {
        setSelectedRelay(hierarchyRes.data.relay_chains[0].relay.name);
      }
      
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

  const getCurrentRelayChain = () => {
    if (!selectedRelay || !hierarchy) return null;
    return hierarchy.relay_chains.find(rc => rc.relay.name === selectedRelay);
  };

  const renderServiceCard = (service, isRelay = false) => {
    const isSelected = selectedService?.name === service.name;
    
    return (
      <div
        key={service.name}
        className={`service-card ${isSelected ? 'selected' : ''} ${isRelay ? 'relay-card' : ''}`}
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

  const currentRelayChain = getCurrentRelayChain();

  return (
    <div className="service-view fade-in">
      <div className="service-header">
        <h1>Service Catalog</h1>
        <div className="view-mode-toggle">
          <button 
            className={`mode-btn ${viewMode === 'hierarchy' ? 'active' : ''}`}
            onClick={() => setViewMode('hierarchy')}
          >
            🏗️ Hierarchy View
          </button>
          <button 
            className={`mode-btn ${viewMode === 'flat' ? 'active' : ''}`}
            onClick={() => setViewMode('flat')}
          >
            📋 List View
          </button>
        </div>
      </div>

      {viewMode === 'hierarchy' ? (
        <div className="hierarchy-view">
          {/* Relay Chains Selector */}
          <div className="relay-selector">
            <h2>Relay Chains</h2>
            <div className="relay-chains-grid">
              {hierarchy?.relay_chains?.map(relayChain => (
                <div
                  key={relayChain.relay.name}
                  className={`relay-selector-card ${selectedRelay === relayChain.relay.name ? 'selected' : ''}`}
                  onClick={() => {
                    setSelectedRelay(relayChain.relay.name);
                    setSelectedService(null);
                  }}
                >
                  {relayChain.relay.logo_url && (
                    <img 
                      src={relayChain.relay.logo_url} 
                      alt={relayChain.relay.display_name}
                      className="relay-selector-logo"
                    />
                  )}
                  <div className="relay-selector-info">
                    <div className="relay-selector-name">{relayChain.relay.display_name}</div>
                    <div className="relay-selector-stats">
                      <span>{relayChain.system_chains?.length || 0} System</span>
                      <span>•</span>
                      <span>{relayChain.community_chains?.length || 0} Community</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Selected Relay Chain Details */}
          {currentRelayChain && (
            <div className="relay-details">
              <div className="relay-info-header">
                <h2>{currentRelayChain.relay.display_name}</h2>
                <button 
                  className="view-relay-btn"
                  onClick={() => setSelectedService(currentRelayChain.relay)}
                >
                  View Relay Details →
                </button>
              </div>

              <div className="chains-container">
                {/* System Chains */}
                <div className="chain-category">
                  <h3>
                    <span className="category-icon">🏛️</span>
                    System Chains ({currentRelayChain.system_chains?.length || 0})
                  </h3>
                  {currentRelayChain.system_chains?.length > 0 ? (
                    <div className="services-grid">
                      {currentRelayChain.system_chains.map(service => 
                        renderServiceCard(service)
                      )}
                    </div>
                  ) : (
                    <div className="no-services">No system chains available</div>
                  )}
                </div>

                {/* Community Chains */}
                <div className="chain-category">
                  <h3>
                    <span className="category-icon">👥</span>
                    Community Chains ({currentRelayChain.community_chains?.length || 0})
                  </h3>
                  {currentRelayChain.community_chains?.length > 0 ? (
                    <div className="services-grid">
                      {currentRelayChain.community_chains.map(service => 
                        renderServiceCard(service)
                      )}
                    </div>
                  ) : (
                    <div className="no-services">No community chains available</div>
                  )}
                </div>
              </div>

              {/* Orphan Services (if any) */}
              {hierarchy?.orphans?.length > 0 && (
                <div className="chain-category">
                  <h3>
                    <span className="category-icon">🔗</span>
                    Other Services ({hierarchy.orphans.length})
                  </h3>
                  <div className="services-grid">
                    {hierarchy.orphans.map(service => 
                      renderServiceCard(service)
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      ) : (
        // Flat view mode - show all services in a list
        <div className="flat-view">
          {/* Original flat view implementation */}
        </div>
      )}

      {/* Service Detail Panel */}
      {selectedService && (
        <div className="service-detail-panel">
          <div className="detail-panel">
            <div className="detail-header">
              <button 
                className="close-detail-btn"
                onClick={() => setSelectedService(null)}
              >
                ✕
              </button>
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
            </div>

            <div className="detail-content">
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
        </div>
      )}
    </div>
  );
};

export default ServiceView;