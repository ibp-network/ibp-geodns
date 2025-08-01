import { getDownServices, getMemberHealth } from './common';

export const getMemberStatus = (member, downtime) => {
  const health = getMemberHealth(member, downtime);
  const hasSiteDowntime = downtime.some(dt =>
    dt.member_name === member.name &&
    dt.check_type === 'site' &&
    !dt.end_time
  );
  
  if (health === 0 || hasSiteDowntime) return 'offline';
  if (health < 100) return 'degraded';
  return 'operational';
};

export const getServiceStatus = (memberName, serviceName, downtime, member) => {
  const memberDowntime = downtime.filter(dt => dt.member_name === memberName);
  const downServices = getDownServices(memberName, member?.services || [], memberDowntime);
  
  if (downServices.has(serviceName)) return 'offline';
  
  const hadDowntime = memberDowntime.some(dt => {
    const eventService = domainToServiceName(dt.domain_name) || dt.endpoint;
    return eventService === serviceName;
  });
  
  return hadDowntime ? 'degraded' : 'operational';
};

// Move domainToServiceName here from common.js
const domainToServiceName = (domainName) => {
  if (!domainName) return null;
  
  let serviceName = domainName
    .replace('.ibp.network', '')
    .replace('.dotters.network', '');
  
  serviceName = serviceName.split('-').map(part =>
    part.charAt(0).toUpperCase() + part.slice(1).toLowerCase()
  ).join('-');
  
  return serviceName;
};