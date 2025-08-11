import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'https://ibdash.dotters.network:9000/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('ibp_auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized
    }
    return Promise.reject(error);
  }
);

const ApiHelper = {
  // Request endpoints
  fetchRequestsByCountry: (params) => api.get('/requests/country', { params }),
  fetchRequestsByASN: (params) => api.get('/requests/asn', { params }),
  fetchRequestsByService: (params) => api.get('/requests/service', { params }),
  fetchRequestsByMember: (params) => api.get('/requests/member', { params }),
  fetchRequestsSummary: (params) => api.get('/requests/summary', { params }),
  
  // Downtime endpoints
  fetchDowntimeEvents: (params) => api.get('/downtime/events', { params }),
  fetchCurrentDowntime: () => api.get('/downtime/current'),
  fetchDowntimeSummary: (params) => api.get('/downtime/summary', { params }),
  
  // Member endpoints
  fetchMembers: () => api.get('/members'),
  fetchMemberStats: (memberName, params) => 
    api.get('/members/stats', { params: { name: memberName, ...params } }),
  
  // Service endpoints
  fetchServices: () => api.get('/services'),
  fetchServicesHierarchy: () => api.get('/services?hierarchy=true'),
  fetchServicesSummary: () => api.get('/services/summary'),
  
  // Billing endpoints
  fetchBillingBreakdown: (params) => api.get('/billing/breakdown', { params }),
  fetchBillingSummary: () => api.get('/billing/summary'),

  // Billing PDF endpoints
  fetchBillingPDFs: (params) => api.get('/billing/pdfs', { params }),
  downloadBillingPDF: (params) => api.get('/billing/pdfs/download', { 
    params,
    responseType: 'arraybuffer'
  }),
};

export default ApiHelper;