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
  getStandardDimensions: (id) => apiRequest(`/standard/${id}/dimensions`),
  getStandardFull: (id) => apiRequest(`/standard/${id}/full`), 
  
  // === ГРУППЫ ===
  getGroups: (queryParams = '') => {
    const url = queryParams ? `/group?${queryParams}` : '/group';
    return apiRequest(url);
  },
  createGroup: (data) => apiRequest('/group', { method: 'POST', body: JSON.stringify(data) }),
  getGroupById: (id) => apiRequest(`/group/${id}`),
  updateGroup: (id, data) => apiRequest(`/group/${id}`, { 
    method: 'PUT', 
    body: JSON.stringify(data) 
  }),
  deleteGroup: (id) => apiRequest(`/group/${id}`, { method: 'DELETE' }),
  
  // === ПРОТОКОЛЫ ===
  getProtocols: (queryParams = '') => {
    const url = queryParams ? `/protocol?${queryParams}` : '/protocol';
    return apiRequest(url);
  },
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
  
  // === ОРГАНИЗАЦИИ ===
  getOrganizations: (limit = 10, offset = 0) => 
    apiRequest(`/organizations?limit=${limit}&offset=${offset}`),
  getOrganizationById: (id) => apiRequest(`/organizations/${id}`),
  getOrganizationByName: (name) => apiRequest(`/organizations/name/${name}`),
  createOrganization: (data) => apiRequest('/organizations', { 
    method: 'POST', 
    body: JSON.stringify(data) 
  }),
  updateOrganization: (id, data) => apiRequest(`/organizations/${id}`, { 
    method: 'PUT', 
    body: JSON.stringify(data) 
  }),
  deleteOrganization: (id) => apiRequest(`/organizations/${id}`, { method: 'DELETE' }),
  
  // === Участники организации ===
  getOrganizationUsers: (orgId, limit = 10, offset = 0) => 
    apiRequest(`/organizations/${orgId}/users?limit=${limit}&offset=${offset}`),
  getOrganizationUserById: (orgId, userId) => 
    apiRequest(`/organizations/${orgId}/users/${userId}`),
  getOrganizationUserByRole: (orgId, role, limit = 10, offset = 0) => 
    apiRequest(`/organizations/${orgId}/users/role/${role}?limit=${limit}&offset=${offset}`),
  createOrganizationUser: (orgId, data) => apiRequest(`/organizations/${orgId}/users`, { 
    method: 'POST', 
    body: JSON.stringify(data) 
  }),
  updateOrganizationUser: (orgId, userId, role) => apiRequest(`/organizations/${orgId}/users/${userId}`, { 
    method: 'PUT', 
    body: JSON.stringify({ role }) 
  }),
  deleteOrganizationUser: (orgId, userId) => 
    apiRequest(`/organizations/${orgId}/users/${userId}`, { method: 'DELETE' }),
  
  // === Тесты/Услуги организации ===
  getOrganizationTests: (limit = 10, offset = 0) => 
    apiRequest(`/organizations/tests?limit=${limit}&offset=${offset}`),
  getOrganizationTest: (testId) => apiRequest(`/organizations/tests/${testId}`),
  createOrganizationTest: (orgId, data) => apiRequest(`/organizations/${orgId}/tests`, { 
    method: 'POST', 
    body: JSON.stringify(data) 
  }),
  updateOrganizationTest: (orgId, testId, data) => apiRequest(`/organizations/${orgId}/tests/${testId}`, { 
    method: 'PUT', 
    body: JSON.stringify(data) 
  }),
  deleteOrganizationTest: (orgId, testId) => 
    apiRequest(`/organizations/${orgId}/tests/${testId}`, { method: 'DELETE' }),
  
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