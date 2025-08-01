import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import './App.css';
import Sidebar from './components/Sidebar/Sidebar';
import DataView from './DataView/DataView';
import EarthView from './EarthView/EarthView';
import MemberView from './MemberView/MemberView';
import MemberDetail from './MemberView/MemberDetail';
import ServiceView from './ServiceView/ServiceView';
import BillingView from './BillingView/BillingView';

function App() {
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
              <Route path="/services" element={<ServiceView />} />
              <Route path="/billing" element={<BillingView />} />
            </Routes>
          </div>
        </div>
      </div>
    </Router>
  );
}

export default App;