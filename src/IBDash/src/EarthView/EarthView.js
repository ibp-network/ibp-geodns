import React, { useEffect, useRef, useState } from 'react';
import Globe from 'globe.gl';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import './EarthView.css';

const EarthView = () => {
  const globeRef = useRef();
  const globeInstance = useRef(null);
  const [members, setMembers] = useState([]);
  const [downtime, setDowntime] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadMembersData();
    loadDowntimeData();
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

  // Calculate member health percentage
  const getMemberHealth = (memberName) => {
    const memberDowntime = downtime.filter(dt => dt.member_name === memberName);
    if (memberDowntime.length === 0) return 100;
    if (memberDowntime.length > 10) return 0;
    return Math.max(0, 100 - (memberDowntime.length * 10));
  };

  useEffect(() => {
    if (!globeRef.current || members.length === 0) return;

    // Clean up previous instance
    if (globeInstance.current) {
      // Properly dispose of the previous globe
      globeInstance.current.scene().children.forEach(child => {
        if (child.geometry) child.geometry.dispose();
        if (child.material) {
          if (child.material.map) child.material.map.dispose();
          child.material.dispose();
        }
      });
      
      if (globeInstance.current.renderer) {
        globeInstance.current.renderer().dispose();
      }
      
      if (globeInstance.current._destructor) {
        globeInstance.current._destructor();
      }
      
      // Clear the container
      while (globeRef.current.firstChild) {
        globeRef.current.removeChild(globeRef.current.firstChild);
      }
      
      globeInstance.current = null;
    }

    // Create new globe instance
    const globe = Globe()(globeRef.current)
      .globeImageUrl('https://unpkg.com/three-globe/example/img/earth-dark.jpg')
      .bumpImageUrl('https://unpkg.com/three-globe/example/img/earth-topology.png')
      .backgroundImageUrl('https://unpkg.com/three-globe/example/img/night-sky.png')
      .showAtmosphere(true)
      .atmosphereColor('lightskyblue')
      .atmosphereAltitude(0.15)
      .pointsData(members)
      .pointLat(d => d.latitude)
      .pointLng(d => d.longitude)
      .pointRadius(0) // Hide the default points
      .pointAltitude(0)
      .htmlElementsData(members)
      .htmlLat(d => d.latitude)
      .htmlLng(d => d.longitude)
      .htmlAltitude(0.01)
      .htmlElement(d => {
        const el = document.createElement('div');
        el.className = 'member-marker';
                 
        const health = getMemberHealth(d.name);
        const status = health === 100 ? 'operational' : health > 50 ? 'degraded' : 'offline';
        
        // Calculate number of active lights (1-5)
        const activeLights = Math.ceil(health / 20);
                 
        // Create member marker with logo
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
                 
        el.style.pointerEvents = 'auto';
        el.style.cursor = 'pointer';
        el.onclick = () => window.location.href = `/members/${d.name}`;
                 
        return el;
      })
      .htmlTransitionDuration(1000);

    // Add connection arcs based on member health
    const arcs = [];
    for (let i = 0; i < members.length; i++) {
      for (let j = i + 1; j < members.length; j++) {
        const health1 = getMemberHealth(members[i].name);
        const health2 = getMemberHealth(members[j].name);
                 
        // Calculate connection probability based on combined health
        const connectionProbability = (health1 + health2) / 200;
                 
        if (Math.random() < connectionProbability * 0.8) {
          // Determine arc color based on average health
          const avgHealth = (health1 + health2) / 2;
          let color;
          
          if (avgHealth >= 80) {
            color = ['rgba(16, 185, 129, 0.6)', 'rgba(16, 185, 129, 0.3)']; // Green
          } else if (avgHealth >= 50) {
            color = ['rgba(245, 158, 11, 0.6)', 'rgba(245, 158, 11, 0.3)']; // Orange
          } else {
            color = ['rgba(239, 68, 68, 0.6)', 'rgba(239, 68, 68, 0.3)']; // Red
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

    // Set up controls
    const controls = globe.controls();
    controls.autoRotate = true;
    controls.autoRotateSpeed = 0.5;
    controls.enableDamping = true;
    controls.dampingFactor = 0.75;
    controls.enableZoom = true;
    controls.zoomSpeed = 0.75;
    controls.minDistance = 150;
    controls.maxDistance = 400;

    // Set initial camera position
    globe.pointOfView({ lat: 20, lng: 0, altitude: 2.5 }, 0);

    // Handle window resize
    const handleResize = () => {
      if (globeRef.current) {
        globe.width(globeRef.current.offsetWidth);
        globe.height(globeRef.current.offsetHeight);
      }
    };
    
    window.addEventListener('resize', handleResize);
    handleResize(); // Initial size

    // Store the instance
    globeInstance.current = globe;

    // Cleanup function
    return () => {
      window.removeEventListener('resize', handleResize);
      
      if (globeInstance.current) {
        // Dispose of globe resources
        if (globeInstance.current.scene) {
          globeInstance.current.scene().children.forEach(child => {
            if (child.geometry) child.geometry.dispose();
            if (child.material) {
              if (child.material.map) child.material.map.dispose();
              child.material.dispose();
            }
          });
        }
        
        if (globeInstance.current.renderer) {
          globeInstance.current.renderer().dispose();
        }
        
        if (globeInstance.current._destructor) {
          globeInstance.current._destructor();
        }
        
        // Clear the container
        while (globeRef.current && globeRef.current.firstChild) {
          globeRef.current.removeChild(globeRef.current.firstChild);
        }
        
        globeInstance.current = null;
      }
    };
  }, [members, downtime]);

  const stats = {
    total: members.length,
    operational: members.filter(m => getMemberHealth(m.name) === 100).length,
    degraded: members.filter(m => {
      const health = getMemberHealth(m.name);
      return health > 0 && health < 100;
    }).length,
    offline: members.filter(m => getMemberHealth(m.name) === 0).length
  };

  if (loading) {
    return (
      <div className="earth-view-loading">
        <div className="loading-spinner"></div>
        <p>Loading global infrastructure map...</p>
      </div>
    );
  }

  return (
    <div className="earth-view fade-in">
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
                     
      <div className="globe-container card">
        <div ref={globeRef} className="globe"></div>
                          
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
            <span>100% Health (5 lights)</span>
          </div>
          <div className="legend-item">
            <div className="legend-health-lights">
              <span className="health-light active"></span>
              <span className="health-light active"></span>
              <span className="health-light active"></span>
              <span className="health-light active"></span>
              <span className="health-light inactive"></span>
            </div>
            <span>80% Health (4 lights)</span>
          </div>
          <div className="legend-item">
            <div className="legend-health-lights">
              <span className="health-light active"></span>
              <span className="health-light active"></span>
              <span className="health-light active"></span>
              <span className="health-light inactive"></span>
              <span className="health-light inactive"></span>
            </div>
            <span>60% Health (3 lights)</span>
          </div>
          <div className="legend-item">
            <div className="legend-health-lights">
              <span className="health-light active"></span>
              <span className="health-light active"></span>
              <span className="health-light inactive"></span>
              <span className="health-light inactive"></span>
              <span className="health-light inactive"></span>
            </div>
            <span>40% Health (2 lights)</span>
          </div>
          <div className="legend-item">
            <div className="legend-health-lights">
              <span className="health-light active"></span>
              <span className="health-light inactive"></span>
              <span className="health-light inactive"></span>
              <span className="health-light inactive"></span>
              <span className="health-light inactive"></span>
            </div>
            <span>20% Health (1 light)</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default EarthView;