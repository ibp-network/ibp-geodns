import React from 'react';
import {
  BarChart, Bar, LineChart, Line, PieChart, Pie,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend,
  ResponsiveContainer, Cell
} from 'recharts';
import './Charts.css';

const COLORS = ['#3b82f6', '#8b5cf6', '#10b981', '#f59e0b', '#ef4444', '#ec4899', '#14b8a6', '#6366f1', '#f472b6', '#a78bfa', '#60a5fa', '#34d399'];

const Charts = ({ data, type }) => {
  if (!data || data.length === 0) return null;

  const CustomTooltip = ({ active, payload }) => {
    if (active && payload && payload.length) {
      return (
        <div className="custom-tooltip">
          {payload.map((entry, index) => (
            <div key={index} className="tooltip-item">
              <span className="tooltip-label">{entry.name}:</span>
              <span className="tooltip-value">{entry.value.toLocaleString()}</span>
            </div>
          ))}
        </div>
      );
    }
    return null;
  };

  const CustomTooltipWithOthers = ({ active, payload, othersData }) => {
    if (active && payload && payload.length) {
      const isOthers = payload[0].payload.service === 'Others';
      return (
        <div className="custom-tooltip">
          {isOthers && othersData ? (
            <>
              <div className="tooltip-item">
                <span className="tooltip-label">Others Total:</span>
                <span className="tooltip-value">{payload[0].value.toLocaleString()}</span>
              </div>
              <div className="tooltip-separator"></div>
              {othersData.map((item, index) => (
                <div key={index} className="tooltip-item">
                  <span className="tooltip-label">{item.service}:</span>
                  <span className="tooltip-value">{item.requests.toLocaleString()}</span>
                </div>
              ))}
            </>
          ) : (
            payload.map((entry, index) => (
              <div key={index} className="tooltip-item">
                <span className="tooltip-label">{entry.name}:</span>
                <span className="tooltip-value">{entry.value.toLocaleString()}</span>
              </div>
            ))
          )}
        </div>
      );
    }
    return null;
  };

  const renderChart = () => {
    switch (type) {
      case 'country':
        // Top 10 countries by requests
        const topCountries = data
          .reduce((acc, item) => {
            const existing = acc.find(c => c.country === item.country);
            if (existing) {
              existing.requests += item.requests;
            } else {
              acc.push({
                country: item.country,
                country_name: item.country_name || item.country,
                requests: item.requests
              });
            }
            return acc;
          }, [])
          .sort((a, b) => b.requests - a.requests)
          .slice(0, 10);

        return (
          <div className="chart-container">
            <h4 className="chart-title">Top 10 Countries by Requests</h4>
            <ResponsiveContainer width="100%" height={400}>
              <BarChart data={topCountries}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis
                  dataKey="country_name"
                  stroke="#666"
                  tick={{ fill: '#999' }}
                  angle={-45}
                  textAnchor="end"
                  height={80}
                />
                <YAxis
                  stroke="#666"
                  tick={{ fill: '#999' }}
                  tickFormatter={(value) => value.toLocaleString()}
                />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="requests" fill={COLORS[0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        );

      case 'asn':
        // Top 10 ASNs
        const topASNs = data
          .reduce((acc, item) => {
            const existing = acc.find(a => a.asn === item.asn);
            if (existing) {
              existing.requests += item.requests;
            } else {
              acc.push({
                asn: item.asn,
                network: item.network,
                requests: item.requests
              });
            }
            return acc;
          }, [])
          .sort((a, b) => b.requests - a.requests)
          .slice(0, 10);

        return (
          <div className="chart-container">
            <h4 className="chart-title">Top 10 Networks by Requests</h4>
            <ResponsiveContainer width="100%" height={400}>
              <BarChart data={topASNs}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis
                  dataKey="network"
                  stroke="#666"
                  tick={{ fill: '#999' }}
                  angle={-45}
                  textAnchor="end"
                  height={100}
                />
                <YAxis
                  stroke="#666"
                  tick={{ fill: '#999' }}
                  tickFormatter={(value) => value.toLocaleString()}
                />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="requests" fill={COLORS[1]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        );

      case 'service':
        // Service distribution pie chart with Others bucket
        const allServiceData = data
          .reduce((acc, item) => {
            // Check both 'service' and 'domain' fields for service names
            const serviceName = item.service || item.domain || 'Unknown';
            const existing = acc.find(s => s.service === serviceName);
            if (existing) {
              existing.requests += item.requests;
            } else {
              acc.push({
                service: serviceName,
                requests: item.requests
              });
            }
            return acc;
          }, [])
          .sort((a, b) => b.requests - a.requests);

        const top10Services = allServiceData.slice(0, 10);
        const othersData = allServiceData.slice(10);
        
        let serviceData = [...top10Services];
        
        if (othersData.length > 0) {
          const othersTotal = othersData.reduce((sum, item) => sum + item.requests, 0);
          serviceData.push({
            service: 'Others',
            requests: othersTotal
          });
        }

        return (
          <div className="chart-container">
            <h4 className="chart-title">Service Distribution</h4>
            <ResponsiveContainer width="100%" height={400}>
              <PieChart>
                <Pie
                  data={serviceData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ service, percent }) => `${service}: ${(percent * 100).toFixed(0)}%`}
                  outerRadius={120}
                  fill="#8884d8"
                  dataKey="requests"
                >
                  {serviceData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.service === 'Others' ? '#6b7280' : COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip content={(props) => <CustomTooltipWithOthers {...props} othersData={othersData} />} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        );

      case 'member':
        // Member distribution pie chart
        const memberData = data
          .reduce((acc, item) => {
            const existing = acc.find(m => m.member === item.member);
            if (existing) {
              existing.requests += item.requests;
            } else {
              acc.push({
                member: item.member,
                requests: item.requests
              });
            }
            return acc;
          }, [])
          .sort((a, b) => b.requests - a.requests)
          .slice(0, 12); // Show top 12 members

        return (
          <div className="chart-container">
            <h4 className="chart-title">Member Distribution</h4>
            <ResponsiveContainer width="100%" height={400}>
              <PieChart>
                <Pie
                  data={memberData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ member, percent }) => `${member}: ${(percent * 100).toFixed(0)}%`}
                  outerRadius={120}
                  fill="#8884d8"
                  dataKey="requests"
                >
                  {memberData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip content={<CustomTooltip />} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        );

      default:
        return null;
    }
  };

  return <div className="charts-wrapper">{renderChart()}</div>;
};

export default Charts;