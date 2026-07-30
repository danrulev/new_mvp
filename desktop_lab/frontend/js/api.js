// frontend/js/api.js
const API_BASE = '/api/v1';

async function apiRequest(endpoint, options = {}) {
  try {
    const res = await fetch(`${API_BASE}${endpoint}`, {
      headers: { 'Content-Type': 'application/json', ...options.headers },
      ...options
    });
    
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }
    
    const ct = res.headers.get('Content-Type');
    if (ct?.includes('application/pdf')) return await res.blob();
    return await res.json();
  } catch (e) {
    console.error(`API ${endpoint}:`, e);
    showToast(e.message, true);
    throw e;
  }
}

export const api = {
  // === МАТЕРИАЛЫ ===
  getMaterials: () => apiRequest('/materials'),
  
  // === СТАНДАРТЫ ===
  getStandardsByMaterial: (id) => apiRequest(`/standard/material/${id}`),
  getMethodsByStandard: (id) => apiRequest(`/standard/${id}/methods`),
  getMethodDetails: (id) => apiRequest(`/standard/method/${id}/details`),
  getStandardDimensions: (id) => apiRequest(`/standard/${id}/dimensions`), // 🔥 Новое
  getStandardFull: (id) => apiRequest(`/standard/${id}/full`), 
  
  // === ГРУППЫ ===
  getGroups: (limit = 50, offset = 0) => apiRequest(`/group?limit=${limit}&offset=${offset}`),
  createGroup: (data) => apiRequest('/group', { method: 'POST', body: JSON.stringify(data) }),
  getGroupById: (id) => apiRequest(`/group/${id}`),
  updateGroup: (id, data) => apiRequest(`/group/${id}`, { 
    method: 'PUT', 
    body: JSON.stringify(data) 
  }),
  deleteGroup: (id) => apiRequest(`/group/${id}`, { method: 'DELETE' }),
  
  // === ПРОТОКОЛЫ ===
  getProtocols: (limit = 20, offset = 0) => apiRequest(`/protocol?limit=${limit}&offset=${offset}`),
  getProtocolFull: (id) => apiRequest(`/protocol/full/${id}`),
  createProtocol: (data) => apiRequest('/protocol', { method: 'POST', body: JSON.stringify(data) }),
  updateProtocolStatus: (id, status) => apiRequest(`/protocol/${id}/status`, { 
    method: 'PUT', body: JSON.stringify({ status }) 
  }),
  getProtocolsByGroup: (id) => apiRequest(`/protocol/groups/${id}`),
  deleteProtocol: (id) => apiRequest(`/protocol/${id}`, { method: 'DELETE' }),
  updateProtocol: (id, data) => apiRequest(`/protocol/${id}`, { 
      method: 'PUT', 
      body: JSON.stringify(data) 
  }),
  
  // === ОТЧЁТЫ ===
  downloadProtocolPDF: (id) => apiRequest(`/report/protocol/${id}/pdf`),
  downloadGroupPDF: (id) => apiRequest(`/report/group/${id}/pdf`),
  
  // === УТИЛИТЫ ===
  request: apiRequest
};

export function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}