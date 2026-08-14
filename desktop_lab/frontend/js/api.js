// frontend/js/api.js
const API_BASE = '/api/v1';

// Флаг для предотвращения рекурсивных вызовов refresh
let isRefreshing = false;
let refreshPromise = null;

/**
 * Получает access токен из localStorage
 */
function getAuthToken() {
  return localStorage.getItem('access_token');
}

// Флаг для предотвращения удаления токена при конкурентных запросах
let isRemovingToken = false;

/**
 * Обновляет access токен используя refresh токен из cookie
 */
async function refreshAccessToken() {
  if (isRefreshing) {
    // Если уже идет процесс обновления, ждем его завершения
    return refreshPromise;
  }
  
  isRefreshing = true;
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'GET',
        credentials: 'include' // Отправляем cookie с refresh токеном
      });
      
      if (!res.ok) {
        throw new Error('Failed to refresh token');
      }
      
      const data = await res.json();
      const newAccessToken = data.access_token;
      
      // Сохраняем новый access токен
      localStorage.setItem('access_token', newAccessToken);
      
      return newAccessToken;
    } catch (error) {
      console.error('Token refresh failed:', error);
      // Ошибка refresh - очищаем данные и редиректим на login
      if (!isRemovingToken) {
        isRemovingToken = true;
        localStorage.removeItem('access_token');
        localStorage.removeItem('user_info');
        isRemovingToken = false;
      }
      if (!window.location.pathname.includes('login.html')) {
        window.location.href = '/login.html';
      }
      throw error;
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  })();
  
  return refreshPromise;
}

async function apiRequest(endpoint, options = {}) {
  try {
    let token = getAuthToken();
    const headers = { 'Content-Type': 'application/json', ...options.headers };
    
    // Добавляем Authorization header если есть токен
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    
    const res = await fetch(`${API_BASE}${endpoint}`, {
      headers,
      ...options
    });
    
    // Обрабатываем 401 Unauthorized - пытаемся обновить токен
    if (res.status === 401) {
      // Если токен истек, пробуем обновить его
      if (!isRefreshing && token) {
        try {
          const newToken = await refreshAccessToken();
          
          // Повторяем исходный запрос с новым токеном
          headers['Authorization'] = `Bearer ${newToken}`;
          const retryRes = await fetch(`${API_BASE}${endpoint}`, {
            headers,
            ...options
          });
          
          // Если повторный запрос успешен, возвращаем результат
          if (retryRes.ok) {
            const ct = retryRes.headers.get('Content-Type');
            if (ct?.includes('application/pdf')) return await retryRes.blob();
            return await retryRes.json();
          }
          
          // Если повторный запрос вернул 401, значит refresh не помог
          if (retryRes.status === 401) {
            if (!isRemovingToken) {
              isRemovingToken = true;
              localStorage.removeItem('access_token');
              localStorage.removeItem('user_info');
              isRemovingToken = false;
            }
            if (!window.location.pathname.includes('login.html')) {
              window.location.href = '/login.html';
            }
            throw new Error('Сессия истекла. Пожалуйста, войдите снова.');
          }
          
          // Обрабатываем другие ошибки повторного запроса
          if (!retryRes.ok) {
            const err = await retryRes.json().catch(() => ({}));
            throw new Error(err.error || err.message || `HTTP ${retryRes.status}`);
          }
        } catch (refreshError) {
          // Ошибка refresh уже обработана в refreshAccessToken
          throw refreshError;
        }
      } else {
        // Token уже удаляется или нет токена - сразу редирект
        if (!isRemovingToken && token) {
          isRemovingToken = true;
          localStorage.removeItem('access_token');
          localStorage.removeItem('user_info');
          isRemovingToken = false;
        }
        
        if (!window.location.pathname.includes('login.html')) {
          window.location.href = '/login.html';
        }
        throw new Error('Сессия истекла. Пожалуйста, войдите снова.');
      }
    }
    
    // Обрабатываем 403 Forbidden
    if (res.status === 403) {
      throw new Error('Доступ запрещен. Недостаточно прав.');
    }
    
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || err.message || `HTTP ${res.status}`);
    }
    
    const ct = res.headers.get('Content-Type');
    if (ct?.includes('application/pdf')) return await res.blob();
    return await res.json();
  } catch (e) {
    console.error(`API ${endpoint}:`, e);
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
  request: apiRequest,
  
  // === ОБРАЗЦЫ (SAMPLES) ===
  createSample: (data) => apiRequest('/sample', { method: 'POST', body: JSON.stringify(data) }),
  getSampleById: (id) => apiRequest(`/sample/${id}`),
  updateSample: (id, data) => apiRequest(`/sample/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteSample: (id) => apiRequest(`/sample/${id}`, { method: 'DELETE' }),
  getSamplesByGroup: (groupID) => apiRequest(`/sample/group/${groupID}`),
  
  // === СОТРУДНИКИ (EMPLOYEES) ===
  getEmployees: () => apiRequest('/employees').then(res => res.employees || [])
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