import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import './App.css';
import Sidebar from './components/Sidebar/Sidebar';
import Header from './components/Header/Header';
import Loading from './components/Loading/Loading';
import DataView from './DataView/DataView';
import EarthView from './EarthView/EarthView';
import MemberView from './MemberView/MemberView';
import MemberDetail from './MemberView/MemberDetail';

function App() {
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setTimeout(() => setLoading(false), 1500);
  }, []);

  if (loading) {
    return <Loading />;
  }

  return (
    <Router>
      <div className="app">
        <Sidebar />
        <div className="main-content">
          <Header />
          <div className="content-area">
            <Routes>
              <Route path="/" element={<Navigate to="/data" replace />} />
              <Route path="/data" element={<DataView />} />
              <Route path="/earth" element={<EarthView />} />
              <Route path="/members" element={<MemberView />} />
              <Route path="/members/:memberName" element={<MemberDetail />} />
            </Routes>
          </div>
        </div>
      </div>
    </Router>
  );
}

export default App;