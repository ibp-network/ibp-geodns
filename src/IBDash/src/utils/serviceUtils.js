export const getServiceTypeIcon = (type) => {
  switch (type?.toUpperCase()) {
    case 'RPC':
      return '🔌';
    case 'ETHRPC':
      return '⟠';
    case 'BOOT':
      return '🚀';
    default:
      return '⚡';
  }
};

export const getServiceTypeLabel = (type) => {
  switch (type?.toUpperCase()) {
    case 'RPC':
      return 'WebSocket RPC';
    case 'ETHRPC':
      return 'Ethereum RPC Proxy';
    case 'BOOT':
      return 'Bootstrap Node';
    default:
      return type || 'Unknown';
  }
};

export const getNetworkTypeIcon = (type) => {
  switch (type?.toLowerCase()) {
    case 'relay':
      return '🏗️';
    case 'system':
      return '🏛️';
    case 'community':
      return '👥';
    default:
      return '🔗';
  }
};

export const getNetworkTypeLabel = (type) => {
  switch (type?.toLowerCase()) {
    case 'relay':
      return 'Relay Chain';
    case 'system':
      return 'System Chain';
    case 'community':
      return 'Community Chain';
    default:
      return type || 'Unknown';
  }
};