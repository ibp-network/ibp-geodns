export const getUptimeClass = (uptime) => {
  if (uptime >= 99.99) return 'excellent';
  if (uptime >= 99.9) return 'good';
  if (uptime >= 99) return 'fair';
  return 'poor';
};

export const getStatusClass = (health) => {
  if (health === 100) return 'operational';
  if (health >= 50) return 'degraded';
  return 'offline';
};

export const domainToServiceName = (domainName) => {
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

export const calculateSiteUptime = (downtimeEvents, startDate, endDate, serviceCount = 0) => {
  if (serviceCount === 0) return 100;
  
  const totalHours = (endDate - startDate) / (1000 * 60 * 60);
  const totalServiceHours = totalHours * serviceCount;
  
  let totalDowntimeServiceHours = 0;
  const currentTime = new Date();
  
  const serviceDowntimeMap = new Map();
  
  downtimeEvents.forEach(event => {
    const eventStart = new Date(event.start_time);
    const eventEnd = event.end_time ? new Date(event.end_time) : currentTime;
    
    const start = eventStart < startDate ? startDate : eventStart;
    const end = eventEnd > endDate ? endDate : eventEnd;
    
    if (start >= end) return;
    
    if (event.check_type === 'site') {
      for (let i = 0; i < serviceCount; i++) {
        const serviceKey = `service_${i}`;
        if (!serviceDowntimeMap.has(serviceKey)) {
          serviceDowntimeMap.set(serviceKey, []);
        }
        serviceDowntimeMap.get(serviceKey).push({ start, end });
      }
    } else {
      const serviceName = domainToServiceName(event.domain_name) || event.endpoint || 'Unknown';
      if (!serviceDowntimeMap.has(serviceName)) {
        serviceDowntimeMap.set(serviceName, []);
      }
      serviceDowntimeMap.get(serviceName).push({ start, end });
    }
  });
  
  serviceDowntimeMap.forEach((periods) => {
    const sortedPeriods = periods.sort((a, b) => a.start - b.start);
    
    const mergedPeriods = [];
    sortedPeriods.forEach(period => {
      if (mergedPeriods.length === 0 || period.start > mergedPeriods[mergedPeriods.length - 1].end) {
        mergedPeriods.push({ ...period });
      } else {
        const lastPeriod = mergedPeriods[mergedPeriods.length - 1];
        lastPeriod.end = period.end > lastPeriod.end ? period.end : lastPeriod.end;
      }
    });
    
    mergedPeriods.forEach(period => {
      totalDowntimeServiceHours += (period.end - period.start) / (1000 * 60 * 60);
    });
  });
  
  const uptimeServiceHours = totalServiceHours - totalDowntimeServiceHours;
  const uptimePercentage = (uptimeServiceHours / totalServiceHours) * 100;
  
  return Math.max(0, Math.min(100, uptimePercentage));
};

export const getDownServices = (memberName, memberServices, downtime) => {
  const memberDowntime = downtime.filter(dt => dt.member_name === memberName);
  const downServices = new Set();
  
  memberDowntime.forEach(dt => {
    if (dt.check_type === 'site') {
      memberServices.forEach(service => downServices.add(service));
    } else if (dt.domain_name) {
      const serviceName = domainToServiceName(dt.domain_name);
      const matchingService = memberServices.find(s => 
        s.toLowerCase() === serviceName?.toLowerCase()
      );
      
      if (matchingService) {
        downServices.add(matchingService);
      }
    }
  });
  
  return downServices;
};

export const getMemberHealth = (member, downtime) => {
  if (!member.services || member.services.length === 0) return 100;
  
  const totalServices = member.services.length;
  const downServices = getDownServices(member.name, member.services, downtime);
  const servicesOnline = totalServices - downServices.size;
  
  return (servicesOnline / totalServices) * 100;
};

export const getStatusIcon = (status) => {
  switch (status) {
    case 'operational': return '✅';
    case 'degraded': return '⚠️';
    case 'offline': return '❌';
    default: return '❓';
  }
};

export const formatMonth = (year, month) => {
  const date = new Date(year, parseInt(month) - 1);
  return date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
};