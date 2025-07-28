import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Globe from 'globe.gl';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import Loading from '../components/Loading/Loading';
import { domainToServiceName, getDownServices, getMemberHealth, getStatusClass } from '../utils/common';
import './EarthView.css';

const EarthView = () => {
  const navigate = useNavigate();
  const globeRef = useRef();
  const containerRef = useRef();
  const globeInstance = useRef(null);
  const [members, setMembers] = useState([]);
  const [downtime, setDowntime] = useState([]);
  const [loading, setLoading] = useState(true);
  const [hoveredMember, setHoveredMember] = useState(null);
  const [pinnedMember, setPinnedMember] = useState(null);
  const panelRef = useRef(null);

  useEffect(() => {
    loadMembersData();
    loadDowntimeData();
  }, []);

  useEffect(() => {
    const handleKeyPress = (e) => {
      if (e.key === 'Escape') {
        setPinnedMember(null);
        setHoveredMember(null);
      }
    };

    window.addEventListener('keydown', handleKeyPress);
    return () => window.removeEventListener('keydown', handleKeyPress);
  }, []);

  const loadMembersData = async () => {
    try {
      const response = await ApiHelper.fetchMembers();
      setMembers(response.data || []);
    } catch (error) {
      console.error('Error loading members:', error);
      setMembers([]);
    }
  };

  const loadDowntimeData = async () => {
    try {
      const response = await ApiHelper.fetchCurrentDowntime();
      setDowntime(response.data || []);
      setLoading(false);
    } catch (error) {
      console.error('Error loading downtime:', error);
      setDowntime([]);
      setLoading(false);
    }
  };

  const getMemberOutages = (memberName) => {
    return downtime.filter(dt => dt.member_name === memberName);
  };

  const isServiceDown = (memberName, serviceName) => {
    const downServices = getDownServices(memberName, members.find(m => m.name === memberName)?.services || [], downtime);
    return downServices.has(serviceName);
  };

  const getServiceStatus = (memberName, serviceName) => {
    return isServiceDown(memberName, serviceName) ? 'offline' : 'online';
  };

  const getTotalDowntimeHours = (memberName) => {
    const outages = getMemberOutages(memberName);
    return outages.length;
  };

  const handleClosePanel = () => {
    setPinnedMember(null);
    setHoveredMember(null);
  };

useEffect(() => {
   if (!containerRef.current || !globeRef.current || members.length === 0) return;

   if (globeInstance.current) {
     if (globeInstance.current._destructor) {
       globeInstance.current._destructor();
     }
     globeInstance.current = null;
   }

   const container = containerRef.current;
   container.addEventListener('dragstart', (e) => e.preventDefault());
   container.addEventListener('selectstart', (e) => e.preventDefault());

   const globe = Globe()(globeRef.current)
     .globeImageUrl('https://unpkg.com/three-globe/example/img/earth-blue-marble.jpg')
     .bumpImageUrl('https://unpkg.com/three-globe/example/img/earth-topology.png')
     .backgroundImageUrl('https://unpkg.com/three-globe/example/img/night-sky.png')
     .showAtmosphere(true)
     .atmosphereColor('lightskyblue')
     .atmosphereAltitude(0.15)
     .pointsData(members)
     .pointLat(d => d.latitude)
     .pointLng(d => d.longitude)
     .pointRadius(0)
     .pointAltitude(0)
     .htmlElementsData(members)
     .htmlLat(d => d.latitude)
     .htmlLng(d => d.longitude)
     .htmlAltitude(0.01)
     .htmlElement(d => {
       const el = document.createElement('div');
       el.className = 'member-marker';
       el.style.pointerEvents = 'auto';
       el.style.cursor = 'pointer';
       
       const health = getMemberHealth(d, downtime);
       const status = getStatusClass(health);
       const activeLights = Math.ceil(health / 20);
       
       el.innerHTML = `
         <div class="marker-container ${status}">
           ${d.logo ?
             `<img src="${d.logo}" alt="${d.name}" class="member-logo-marker" onerror="this.onerror=null; this.parentElement.innerHTML='<div class=\\'member-logo-placeholder\\'>${d.name.substring(0, 2).toUpperCase()}</div>'" />` :
             `<div class="member-logo-placeholder">${d.name.substring(0, 2).toUpperCase()}</div>`
           }
           <div class="member-name-label">${d.name}</div>
           <div class="health-lights">
             ${Array.from({ length: 5 }, (_, i) =>
               `<span class="health-light ${i < activeLights ? 'active' : 'inactive'}"></span>`
             ).join('')}
           </div>
         </div>
       `;
       
       el.onmouseenter = () => setHoveredMember(d);
       el.onclick = (e) => {
         e.stopPropagation();
         navigate(`/members/${d.name}`);
       };
       
       return el;
     })
     .htmlTransitionDuration(1000);

   const arcs = [];
   for (let i = 0; i < members.length; i++) {
     for (let j = i + 1; j < members.length; j++) {
       const health1 = getMemberHealth(members[i], downtime);
       const health2 = getMemberHealth(members[j], downtime);
       const connectionProbability = (health1 + health2) / 200;
       
       if (Math.random() < connectionProbability * 0.8) {
         const avgHealth = (health1 + health2) / 2;
         let color;
         if (avgHealth >= 80) {
           color = ['rgba(16, 185, 129, 0.6)', 'rgba(16, 185, 129, 0.3)'];
         } else if (avgHealth >= 50) {
           color = ['rgba(245, 158, 11, 0.6)', 'rgba(245, 158, 11, 0.3)'];
         } else {
           color = ['rgba(239, 68, 68, 0.6)', 'rgba(239, 68, 68, 0.3)'];
         }
         
         arcs.push({
           startLat: members[i].latitude,
           startLng: members[i].longitude,
           endLat: members[j].latitude,
           endLng: members[j].longitude,
           color: color
         });
       }
     }
   }

   globe
     .arcsData(arcs)
     .arcColor('color')
     .arcDashLength(0.4)
     .arcDashGap(0.2)
     .arcDashAnimateTime(2000)
     .arcStroke(0.5)
     .arcAltitudeAutoScale(0.3);

   const controls = globe.controls();
   controls.autoRotate = false;
   controls.enableDamping = true;
   controls.dampingFactor = 0.75;
   controls.enableZoom = true;
   controls.zoomSpeed = 0.75;
   controls.minDistance = 150;
   controls.maxDistance = 400;

   globe.pointOfView({ lat: 20, lng: 0, altitude: 2.5 }, 0);

   const handleResize = () => {
     if (containerRef.current && globeRef.current) {
       const { width, height } = containerRef.current.getBoundingClientRect();
       globe.width(width);
       globe.height(height);
     }
   };

   handleResize();
   window.addEventListener('resize', handleResize);
   setTimeout(handleResize, 100);

   globeInstance.current = globe;

   return () => {
     container.removeEventListener('dragstart', (e) => e.preventDefault());
     container.removeEventListener('selectstart', (e) => e.preventDefault());
     window.removeEventListener('resize', handleResize);
     if (globeInstance.current && globeInstance.current._destructor) {
       globeInstance.current._destructor();
     }
     globeInstance.current = null;
   };
 }, [members, downtime, navigate]);

 const stats = {
   total: members.length,
   operational: members.filter(m => getMemberHealth(m, downtime) === 100).length,
   degraded: members.filter(m => {
     const health = getMemberHealth(m, downtime);
     return health > 0 && health < 100;
   }).length,
   offline: members.filter(m => getMemberHealth(m, downtime) === 0).length
 };

 const displayMember = pinnedMember || hoveredMember;

 if (loading) {
   return <Loading pageLevel={true} dataReady={false} />;
 }

 return (
   <div className="earth-view fade-in">
     <div className="globe-container" ref={containerRef}>
       <div className="globe-wrapper">
         <div ref={globeRef} className="globe"></div>
       </div>
       
       <div className="earth-header">
         <h1>Global Infrastructure Map</h1>
         <div className="status-summary enhanced-glass">
           <div className="status-item">
             <span className="status-indicator status-online"></span>
             <span className="status-label">{stats.operational} Operational</span>
           </div>
           <div className="status-item">
             <span className="status-indicator status-warning"></span>
             <span className="status-label">{stats.degraded} Degraded</span>
           </div>
           <div className="status-item">
             <span className="status-indicator status-offline"></span>
             <span className="status-label">{stats.offline} Offline</span>
           </div>
         </div>
       </div>

       <div
         ref={panelRef}
         className={`member-info-panel enhanced-glass ${displayMember ? 'visible' : ''} ${pinnedMember ? 'pinned' : ''}`}
       >
         {displayMember && (
           <>
             <div className="panel-close-btn" onClick={handleClosePanel}>✕</div>
             <div className="panel-header">
               {displayMember.logo ? (
                 <img src={displayMember.logo} alt={displayMember.name} className="panel-logo" />
               ) : (
                 <div className="panel-logo logo-placeholder">
                   {displayMember.name.substring(0, 2).toUpperCase()}
                 </div>
               )}
               <div className="panel-title">
                 <div className="panel-name">{displayMember.name}</div>
                 <div className="panel-region">{displayMember.region}</div>
               </div>
             </div>
             <div className="panel-content">
               <div className="info-section">
                 <h3 className="section-title">Member Information</h3>
                 <div className="info-grid compact">
                   <div className="info-row">
                     <span className="info-label">Health:</span>
                     <span className="info-value">{getMemberHealth(displayMember, downtime).toFixed(0)}%</span>
                   </div>
                   <div className="info-row">
                     <span className="info-label">Level:</span>
                     <span className="info-value">{displayMember.level}</span>
                   </div>
                   <div className="info-row">
                     <span className="info-label">IPv4:</span>
                     <span className="info-value">{displayMember.service_ipv4 || 'N/A'}</span>
                   </div>
                   <div className="info-row">
                     <span className="info-label">IPv6:</span>
                     <span className="info-value">{displayMember.service_ipv6 || 'N/A'}</span>
                   </div>
                   <div className="info-row">
                     <span className="info-label">Location:</span>
                     <span className="info-value">{displayMember.latitude?.toFixed(2)}, {displayMember.longitude?.toFixed(2)}</span>
                   </div>
                   <div className="info-row">
                     <span className="info-label">Downtime:</span>
                     <span className="info-value">{getTotalDowntimeHours(displayMember.name)}h</span>
                   </div>
                   <div className="info-row full-width">
                     <span className="info-label">Website:</span>
                     <a href={displayMember.website} target="_blank" rel="noopener noreferrer" className="info-value link">
                       {displayMember.website?.replace(/^https?:\/\//, '')}
                     </a>
                   </div>
                 </div>
               </div>

               {getMemberOutages(displayMember.name).length > 0 && (
                 <div className="active-events">
                   <div className="active-events-title">Active Events</div>
                   {getMemberOutages(displayMember.name).slice(0, 3).map((outage, idx) => (
                     <div key={idx} className="event-item">
                       <span className="event-type">{outage.check_type}:</span>
                       <span>{outage.domain_name || outage.endpoint || 'Site level issue'}</span>
                     </div>
                   ))}
                   {getMemberOutages(displayMember.name).length > 3 && (
                     <div className="event-item">
                       <span>...and {getMemberOutages(displayMember.name).length - 3} more</span>
                     </div>
                   )}
                 </div>
               )}

               {displayMember.services && displayMember.services.length > 0 && (
                 <div className="info-section">
                   <h3 className="section-title">Active Services</h3>
                   <div className="services-grid">
                     {displayMember.services.map((service, idx) => {
                       const status = getServiceStatus(displayMember.name, service);
                       return (
                         <div key={idx} className="service-item">
                           <span className="service-name">{service}</span>
                           <div className="service-status">
                             <span className={`status-dot ${status}`}></span>
                             <span>{status.charAt(0).toUpperCase() + status.slice(1)}</span>
                           </div>
                         </div>
                       );
                     })}
                   </div>
                 </div>
               )}
             </div>
           </>
         )}
       </div>

       <div className="globe-controls enhanced-glass">
         <h3>Controls</h3>
         <div className="control-item">
           <span className="control-icon">🖱️</span>
           <span>Drag to rotate</span>
         </div>
         <div className="control-item">
           <span className="control-icon">👆</span>
           <span>Click member for details</span>
         </div>
         <div className="control-item">
           <span className="control-icon">🔍</span>
           <span>Scroll to zoom</span>
         </div>
       </div>

       <div className="globe-legend enhanced-glass">
         <h3>Member Status</h3>
         <div className="legend-item">
           <div className="legend-health-lights">
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light active"></span>
           </div>
           <span>100% Services Online</span>
         </div>
         <div className="legend-item">
           <div className="legend-health-lights">
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light inactive"></span>
           </div>
           <span>80% Services Online</span>
         </div>
         <div className="legend-item">
           <div className="legend-health-lights">
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light inactive"></span>
             <span className="health-light inactive"></span>
           </div>
           <span>60% Services Online</span>
         </div>
         <div className="legend-item">
           <div className="legend-health-lights">
             <span className="health-light active"></span>
             <span className="health-light active"></span>
             <span className="health-light inactive"></span>
             <span className="health-light inactive"></span>
             <span className="health-light inactive"></span>
           </div>
           <span>40% Services Online</span>
         </div>
         <div className="legend-item">
           <div className="legend-health-lights">
             <span className="health-light active"></span>
             <span className="health-light inactive"></span>
             <span className="health-light inactive"></span>
             <span className="health-light inactive"></span>
             <span className="health-light inactive"></span>
           </div>
           <span>20% Services Online</span>
         </div>
       </div>
     </div>
   </div>
 );
};

export default EarthView;