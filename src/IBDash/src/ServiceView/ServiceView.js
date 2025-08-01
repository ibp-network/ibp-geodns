import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Loading from '../components/Loading/Loading';
import { getServiceTypeIcon, getServiceTypeLabel } from '../utils/serviceUtils';
import './ServiceView.css';

const ServiceView = () => {
  const navigate = useNavigate();
  const [services, setServices] = useState([]);
  const [members, setMembers] = useState([]);
  const [selectedService, setSelectedService] = useState(null);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [copiedEndpoint, setCopiedEndpoint] = useState(null);

  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    try {
      const [servicesRes, membersRes] = await Promise.all([
        ApiHelper.fetchServices(),
        ApiHelper.fetchMembers()
      ]);
      
      setServices(servicesRes.data?.services || []);
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
          },
          {
            label: 'wscat Example:',
            code: `wscat -c wss://${service.providers[0]?.rpc_urls[0]?.replace('wss://', '') || 'example.com'}`
          }
        ]
      };
    } else if (service.service_type === 'ETHRPC') {
      return {
        title: 'Ethereum RPC Proxy Connection',
        description: 'Connect using Ethereum-compatible tools to access Polkadot chains via Revival Networks proxy:',
        examples: [
          {
            label: 'Web3.js Example:',
            code: `import Web3 from 'web3';

const web3 = new Web3('wss://${service.providers[0]?.rpc_urls[0]?.replace('wss://', '') || 'example.com'}');

// Get chain ID
const chainId = await web3.eth.getChainId();
console.log('Chain ID:', chainId);`
          },
          {
            label: 'ethers.js Example:',
            code: `import { ethers } from 'ethers';

const provider = new ethers.providers.WebSocketProvider('wss://${service.providers[0]?.rpc_urls[0]?.replace('wss://', '') || 'example.com'}');
const network = await provider.getNetwork();
console.log('Network:', network);`
          }
        ]
      };
    } else if (service.service_type === 'BOOT') {
      return {
        title: 'Bootstrap Node Usage',
        description: 'Use this bootstrap node to join the network:',
        examples: [
          {
            label: 'Node Configuration:',
            code: `./polkadot --chain=${service.network_name?.toLowerCase() || 'polkadot'} \\
  --bootnodes="/dns/${service.providers[0]?.rpc_urls[0]?.replace('wss://', '').replace('ws://', '') || 'example.com'}/tcp/30333/p2p/PEER_ID"`
          }
        ]
      };
    }
    return null;
  };

  const filteredServices = services
    .filter(service => 
      service.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      service.display_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      service.network_name?.toLowerCase().includes(searchTerm.toLowerCase())
    )
    .sort((a, b) => a.display_name?.localeCompare(b.display_name));

  if (loading) {
    return <Loading pageLevel={true} dataReady={true} />;
  }

  return (
    <div className="service-view fade-in">
      <div className="service-header">
        <h1>Service Catalog</h1>
      </div>

      <div className="services-nav-bar">
        <div className="services-nav-header">
          <h2>Available Services</h2>
          <input
            type="text"
            placeholder="Search services..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="service-search"
          />
        </div>
        <div className="services-grid-list">
          {filteredServices.map(service => (
            <div
              key={service.name}
              className={`service-nav-item ${selectedService?.name === service.name ? 'active' : ''}`}
              onClick={() => setSelectedService(service)}
            >
              {service.logo_url ? (
                <img
                  src={service.logo_url} 
                  alt={service.display_name} 
                  className="service-logo-small"
                  onError={(e) => {
                    e.target.style.display = 'none';
                    e.target.nextSibling.style.display = 'flex';
                  }}
                />
              ) : null}
              <div 
                className="service-logo-placeholder" 
                style={{ display: service.logo_url ? 'none' : 'flex' }}
              >
                {service.display_name?.substring(0, 2).toUpperCase()}
              </div>
              <div className="service-nav-name">{service.display_name || service.name}</div>
              <div className={`service-type-indicator ${service.service_type?.toLowerCase()}`} 
                   title={getServiceTypeLabel(service.service_type)}></div>
            </div>
          ))}
        </div>
      </div>

      <div className="detail-panel">
        {selectedService ? (
          <>
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
                  <span>{selectedService.network_name || 'Unknown Network'}</span>
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
          </>
        ) : (
          <div className="placeholder-content">
            <div className="placeholder-icon">⚡</div>
            <h3>Select a service to view details</h3>
            <p>Choose from the list above to see service information and usage instructions</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ServiceView;