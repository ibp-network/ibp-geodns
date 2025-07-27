import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import './App.css';
import Sidebar from './components/Sidebar/Sidebar';
import Loading from './components/Loading/Loading';
import DataView from './DataView/DataView';
import EarthView from './EarthView/EarthView';
import MemberView from './MemberView/MemberView';
import MemberDetail from './MemberView/MemberDetail';
import BillingView from './BillingView/BillingView';

function App() {
  const [loading, setLoading] = useState(true);
  const [dataReady, setDataReady] = useState(false);

  useEffect(() => {
    // Simulate data loading - in real app, this would be your API calls
    const loadData = async () => {
      // Your actual data loading logic here
      await new Promise(resolve => setTimeout(resolve, 500)); // Simulated API call
      setDataReady(true);
    };

    loadData();
  }, []);

  const handleLoadingComplete = () => {
    if (dataReady) {
      setLoading(false);
    }
  };

  if (loading) {
    return <Loading onAnimationComplete={handleLoadingComplete} dataReady={dataReady} />;
  }

  return (
    <Router>
      <div className="app">
        <Sidebar />
        <div className="main-content">
          <div className="content-area">
            <Routes>
              <Route path="/" element={<Navigate to="/data" replace />} />
              <Route path="/data" element={<DataView />} />
              <Route path="/earth" element={<EarthView />} />
              <Route path="/members" element={<MemberView />} />
              <Route path="/members/:memberName" element={<MemberDetail />} />
              <Route path="/billing" element={<BillingView />} />
            </Routes>
          </div>
        </div>
      </div>
    </Router>
  );
}

export default App;