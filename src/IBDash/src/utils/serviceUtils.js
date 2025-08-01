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