// frontend/js/auth.js - Модуль аутентификации и управления сессией

const AUTH_STORAGE_KEY = 'access_token';
const USER_INFO_KEY = 'user_info';

export const auth = {
  /**
   * Проверяет, авторизован ли пользователь
   */
  isAuthenticated() {
    return !!localStorage.getItem(AUTH_STORAGE_KEY);
  },

  /**
   * Получает access токен
   */
  getToken() {
    return localStorage.getItem(AUTH_STORAGE_KEY);
  },

  /**
   * Сохраняет access токен
   */
  setToken(token) {
    localStorage.setItem(AUTH_STORAGE_KEY, token);
  },

  /**
   * Удаляет токен (выход)
   */
  logout() {
    localStorage.removeItem(AUTH_STORAGE_KEY);
    localStorage.removeItem(USER_INFO_KEY);
  },

  /**
   * Получает информацию о пользователе
   */
  getUserInfo() {
    const userInfo = localStorage.getItem(USER_INFO_KEY);
    return userInfo ? JSON.parse(userInfo) : null;
  },

  /**
   * Сохраняет информацию о пользователе
   */
  setUserInfo(user) {
    localStorage.setItem(USER_INFO_KEY, JSON.stringify(user));
  },

  /**
   * Получает роль пользователя
   */
  getRole() {
    const user = this.getUserInfo();
    return user?.role || null;
  },

  /**
   * Проверяет, есть ли у пользователя определенная роль
   */
  hasRole(requiredRole) {
    const userRole = this.getRole();
    if (!userRole) return false;
    
    // Иерархия ролей: admin > engineer > technician > client
    const roleHierarchy = {
      'admin': ['admin', 'engineer', 'technician', 'client'],
      'engineer': ['engineer', 'technician', 'client'],
      'technician': ['technician', 'client'],
      'client': ['client']
    };
    
    return roleHierarchy[requiredRole]?.includes(userRole) || false;
  },

  /**
   * Проверяет, является ли пользователь админом
   */
  isAdmin() {
    return this.getRole() === 'admin';
  },

  /**
   * Проверяет, является ли пользователь инженером или админом
   */
  isEngineer() {
    const role = this.getRole();
    return role === 'engineer' || role === 'admin';
  },

  /**
   * Проверяет, является ли пользователь техником или выше
   */
  isTechnician() {
    const role = this.getRole();
    return role === 'technician' || role === 'engineer' || role === 'admin';
  },

  /**
   * Проверяет, является ли пользователь клиентом
   */
  isClient() {
    return this.getRole() === 'client';
  },

  /**
   * Выполняет вход
   */
  async login(email, password) {
    try {
      const response = await fetch('/api/v1/auth/sign-in', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      });
      
      if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.message || err.error || 'Ошибка входа');
      }
      
      const data = await response.json();
      this.setToken(data.access_token);
      
      // Загружаем информацию о пользователе
      await this.loadUserInfo();
      
      return { success: true };
    } catch (error) {
      console.error('Login error:', error);
      return { success: false, error: error.message };
    }
  },

  /**
   * Выполняет регистрацию
   */
  async register(name, email, password, role) {
    try {
      const response = await fetch('/api/v1/auth/sign-up', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, email, password, role })
      });
      
      if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.message || err.error || 'Ошибка регистрации');
      }
      
      return { success: true };
    } catch (error) {
      console.error('Registration error:', error);
      return { success: false, error: error.message };
    }
  },

  /**
   * Выполняет выход
   */
  async logout() {
    try {
      const token = this.getToken();
      if (token) {
        await fetch('/api/v1/auth/logout', {
          headers: { 'Authorization': `Bearer ${token}` }
        });
      }
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      this.logout();
      window.location.href = '/login.html';
    }
  },

  /**
   * Загружает информацию о пользователе
   */
  async loadUserInfo() {
    try {
      const token = this.getToken();
      if (!token) return null;
      
      const response = await fetch('/api/v1/auth/me', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (response.ok) {
        const user = await response.json();
        this.setUserInfo(user);
        return user;
      }
      
      return null;
    } catch (error) {
      console.error('Failed to load user info:', error);
      return null;
    }
  },

  /**
   * Проверяет валидность токена и загружает данные пользователя
   */
  async validateSession() {
    if (!this.isAuthenticated()) {
      return false;
    }
    
    const user = await this.loadUserInfo();
    return !!user;
  },

  /**
   * Требует аутентификацию - редиректит на login если не авторизован
   */
  requireAuth() {
    if (!this.isAuthenticated()) {
      window.location.href = '/login.html';
      return false;
    }
    return true;
  },

  /**
   * Обновляет UI в зависимости от роли пользователя
   */
  updateUIByRole() {
    const role = this.getRole();
    if (!role) return;
    
    // Скрываем/показываем элементы по ролям
    document.querySelectorAll('[data-requires-role]').forEach(el => {
      const requiredRoles = el.dataset.requiresRole.split(',');
      const hasAccess = requiredRoles.some(r => this.hasRole(r.trim()));
      el.style.display = hasAccess ? '' : 'none';
    });
    
    // Показываем элементы только для определенных ролей
    document.querySelectorAll('[data-show-for-role]').forEach(el => {
      const allowedRoles = el.dataset.showForRole.split(',');
      const hasAccess = allowedRoles.some(r => this.getRole() === r.trim());
      el.style.display = hasAccess ? '' : 'none';
    });
    
    // Отображаем имя пользователя и роль
    const userInfo = this.getUserInfo();
    if (userInfo) {
      const userNameEls = document.querySelectorAll('.user-name');
      userNameEls.forEach(el => {
        el.textContent = userInfo.name || userInfo.email;
      });
      
      const userRoleEls = document.querySelectorAll('.user-role');
      userRoleEls.forEach(el => {
        el.textContent = this.getRoleDisplayName(role);
      });
    }
  },

  /**
   * Возвращает отображаемое имя роли
   */
  getRoleDisplayName(role) {
    const roleNames = {
      'admin': 'Администратор',
      'engineer': 'Инженер',
      'technician': 'Техник',
      'client': 'Клиент'
    };
    return roleNames[role] || role;
  },

  /**
   * Инициализирует аутентификацию при загрузке страницы
   */
  init() {
    // Проверяем сессию при загрузке
    this.validateSession().then(isValid => {
      if (!isValid && !window.location.pathname.includes('login.html') && !window.location.pathname.includes('register.html')) {
        // Не редиректим сразу, даем странице загрузиться
        console.log('Session invalid, will redirect to login');
      }
    });
    
    // Обновляем UI по ролям
    this.updateUIByRole();
  }
};

// Экспортируем вспомогательные функции для проверки прав
export function canCreateProtocol() {
  return auth.isTechnician();
}

export function canEditProtocol() {
  return auth.isEngineer();
}

export function canDeleteProtocol() {
  return auth.isEngineer();
}

export function canViewReports() {
  return auth.isAuthenticated();
}

export function canCreateUsers() {
  return auth.isAdmin();
}

export function canManageOrganization() {
  return auth.isAdmin() || auth.isEngineer();
}
