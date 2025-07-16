import React, { useEffect, useRef, useState } from 'react';
import Globe from 'globe.gl';
import ApiHelper from '../components/ApiHelper/ApiHelper';
import './EarthView.css';

const EarthView = () => {
  const globeRef = useRef();
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
      setMembers(response.data);
    } catch (error) {
      console.error('Error loading members:', error);
    }
  };

  const loadDowntimeData = async () => {
    try {
      const response = await ApiHelper.fetchCurrentDowntime();
      setDowntime(response.data);
      setLoading(false);
    } catch (error) {
      console.error('Error loading downtime:', error);
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!globeRef.current || members.length === 0) return;

    const globe = Globe()(globeRef.current)
      .globeImageUrl('//unpkg.com/three-globe/example/img/earth-dark.jpg')
      .backgroundImageUrl('//unpkg.com/three-globe/example/img/night-sky.png')
      .pointsData(members)
      .pointLat(d => d.latitude)
      .pointLng(d => d.longitude)
      .pointColor(d => {
        const memberDowntime = downtime.filter(dt => dt.member_name === d.name);
        if (memberDowntime.length > 5) return '#ef4444'; // Red - major issues
        if (memberDowntime.length > 0) return '#f59e0b'; // Orange - some issues
        return '#10b981'; // Green - all good
      })
      .pointRadius(d => {
        const memberDowntime = downtime.filter(dt => dt.member_name === d.name);
        return memberDowntime.length > 0 ? 0.8 : 0.6;
      })
      .pointAltitude(0.01)
      .pointLabel(d => {
        const memberDowntime = downtime.filter(dt => dt.member_name === d.name);
        const status = memberDowntime.length === 0 ? 'Operational' : 
                      memberDowntime.length > 5 ? 'Major Outage' : 'Degraded';
        return `
          <div style="text-align: center; padding: 8px; background: rgba(0,0,0,0.8); border-radius: 8px;">
            <div style="font-weight: bold; font-size: 14px; color: #fff;">${d.name}</div>
            <div style="font-size: 12px; color: #888; margin: 4px 0;">${d.region}</div>
            <div style="font-size: 12px; margin-top: 4px;">
              Status: <span style="color: ${
                status === 'Operational' ? '#10b981' : 
                status === 'Major Outage' ? '#ef4444' : '#f59e0b'
              }; font-weight: bold;">${status}</span>
            </div>
            ${memberDowntime.length > 0 ? 
              `<div style="font-size: 11px; color: #f59e0b; margin-top: 4px;">
                ${memberDowntime.length} service(s) affected
              </div>` : ''
            }
            ${d.service_ipv4 ? `<div style="font-size: 10px; color: #666; margin-top: 4px;">IPv4: ${d.service_ipv4}</div>` : ''}
            ${d.service_ipv6 ? `<div style="font-size: 10px; color: #666;">IPv6: ${d.service_ipv6}</div>` : ''}
          </div>
        `;
      })
      .onPointClick(point => {
        window.location.href = `/members/${point.name}`;
      })
      .onPointHover(point => {
        document.body.style.cursor = point ? 'pointer' : 'default';
      });

    // Add connection arcs
    const arcs = [];
    for (let i = 0; i < members.length; i++) {
      for (let j = i + 1; j < members.length; j++) {
        if (Math.random() > 0.85) { // Show only some connections
          arcs.push({
            startLat: members[i].latitude,
            startLng: members[i].longitude,
            endLat: members[j].latitude,
            endLng: members[j].longitude,
            color: ['rgba(59, 130, 246, 0.5)', 'rgba(139, 92, 246, 0.5)']
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
      .arcStroke(0.3)
      .arcAltitudeAutoScale(0.3);

    // Auto-rotate
    globe.controls().autoRotate = true;
    globe.controls().autoRotateSpeed = 0.5;

    // Set initial position
    globe.pointOfView({ lat: 20, lng: 0, altitude: 2.5 });

    // Add atmosphere
    const globeEl = globe.domElement;
    globeEl.style.background = 'radial-gradient(circle at 50% 50%, #1a1a2e 0%, #0a0a0a 100%)';

    return () => {
      if (globe) {
        globe._destructor();
      }
    };
  }, [members, downtime]);

  const stats = {
    total: members.length,
    operational: members.filter(m => !downtime.find(d => d.member_name === m.name)).length,
    degraded: members.filter(m => {
      const dt = downtime.filter(d => d.member_name === m.name);
      return dt.length > 0 && dt.length <= 5;
    }).length,
    offline: members.filter(m => {
      const dt = downtime.filter(d => d.member_name === m.name);
      return dt.length > 5;
    }).length
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
        <div className="status-summary glass">
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
        
        <div className="globe-controls glass">
          <h3>Controls</h3>
          <div className="control-item">
            <span className="control-icon">🖱️</span>
            <span>Drag to rotate</span>
          </div>
          <div className="control-item">
            <span className="control-icon">📍</span>
            <span>Click member for details</span>
          </div>
          <div className="control-item">
            <span className="control-icon">🔍</span>
            <span>Scroll to zoom</span>
          </div>
        </div>

        <div className="globe-legend glass">
          <h3>Member Status</h3>
          <div className="legend-item">
            <span className="legend-dot operational"></span>
            <span>Fully Operational</span>
          </div>
          <div className="legend-item">
            <span className="legend-dot degraded"></span>
            <span>Degraded Performance</span>
          </div>
          <div className="legend-item">
            <span className="legend-dot offline"></span>
            <span>Major Outage</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default EarthView;