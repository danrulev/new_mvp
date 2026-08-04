// frontend/js/organizations.js
import { api } from './api.js';
import { formatDate, showToast, showConfirm } from './utils.js';
import { initNavigation } from './navigation.js';

let currentOrgId = null;
let currentOffset = 0;
const limit = 10;

// Инициализация при загрузке
document.addEventListener('DOMContentLoaded', () => {
  initNavigation();
  loadOrganizations();
});

// Загрузка организаций
async function loadOrganizations(offset = 0) {
  try {
    const data = await api.getOrganizations(limit, offset);
    renderOrganizations(data.organizations || []);
    renderPagination(data.meta || { total: 0, limit, offset });
  } catch (error) {
    console.error('Ошибка загрузки организаций:', error);
    document.getElementById('organizationsTableBody').innerHTML = `
      <tr><td colspan="7" class="text-center text-muted">Ошибка загрузки: ${error.message}</td></tr>
    `;
  }
}

// Рендеринг таблицы организаций
function renderOrganizations(organizations) {
  const tbody = document.getElementById('organizationsTableBody');
  
  if (!organizations || organizations.length === 0) {
    tbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted">Организации не найдены</td></tr>';
    return;
  }
  
  tbody.innerHTML = organizations.map(org => `
    <tr>
      <td><strong>${escapeHtml(org.name)}</strong></td>
      <td>${escapeHtml(org.address)}</td>
      <td>${escapeHtml(org.phone)}</td>
      <td><a href="mailto:${escapeHtml(org.email)}">${escapeHtml(org.email)}</a></td>
      <td>
        <button class="btn btn-secondary btn-small" onclick="openUsersModal('${org.id}', '${escapeHtml(org.name)}')">
          👥 Участники
        </button>
      </td>
      <td>
        <button class="btn btn-secondary btn-small" onclick="openTestsModal('${org.id}', '${escapeHtml(org.name)}')">
          🧪 Услуги
        </button>
      </td>
      <td>
        <div class="actions">
          <button class="btn btn-primary btn-small" onclick="editOrganization('${org.id}')">✏️</button>
          <button class="btn btn-danger btn-small" onclick="deleteOrganization('${org.id}')">🗑️</button>
        </div>
      </td>
    </tr>
  `).join('');
}

// Рендеринг пагинации
function renderPagination(meta) {
  const pagination = document.getElementById('pagination');
  const totalPages = Math.ceil((meta.total || 0) / limit);
  const currentPage = Math.floor((meta.offset || 0) / limit) + 1;
  
  if (totalPages <= 1) {
    pagination.innerHTML = '';
    return;
  }
  
  let html = '<div style="display: flex; justify-content: center; gap: 8px; margin-top: 24px;">';
  
  // Предыдущая
  if (currentPage > 1) {
    html += `<button class="btn btn-secondary btn-small" onclick="loadOrganizations(${(currentPage - 2) * limit})">← Назад</button>`;
  }
  
  // Страницы
  for (let i = 1; i <= totalPages; i++) {
    if (i === currentPage) {
      html += `<button class="btn btn-primary btn-small">${i}</button>`;
    } else {
      html += `<button class="btn btn-secondary btn-small" onclick="loadOrganizations(${(i - 1) * limit})">${i}</button>`;
    }
  }
  
  // Следующая
  if (currentPage < totalPages) {
    html += `<button class="btn btn-secondary btn-small" onclick="loadOrganizations(${currentPage * limit})">Вперёд →</button>`;
  }
  
  html += '</div>';
  pagination.innerHTML = html;
}

// Поиск
function handleSearch() {
  const query = document.getElementById('searchInput').value.toLowerCase().trim();
  // Пока просто фильтруем на клиенте - в будущем можно добавить серверный поиск
  const rows = document.querySelectorAll('#organizationsTableBody tr');
  rows.forEach(row => {
    const text = row.textContent.toLowerCase();
    row.style.display = text.includes(query) ? '' : 'none';
  });
}

// Открытие модального окна создания
function openCreateModal() {
  document.getElementById('modalTitle').textContent = 'Новая организация';
  document.getElementById('orgForm').reset();
  document.getElementById('orgId').value = '';
  document.getElementById('orgModal').classList.add('active');
}

// Закрытие модального окна
function closeModal() {
  document.getElementById('orgModal').classList.remove('active');
}

// Редактирование организации
async function editOrganization(id) {
  try {
    const org = await api.getOrganizationById(id);
    document.getElementById('modalTitle').textContent = 'Редактирование организации';
    document.getElementById('orgId').value = org.id;
    document.getElementById('orgName').value = org.name;
    document.getElementById('orgAddress').value = org.address;
    document.getElementById('orgPhone').value = org.phone;
    document.getElementById('orgEmail').value = org.email;
    document.getElementById('orgModal').classList.add('active');
  } catch (error) {
    showToast('Ошибка загрузки данных организации: ' + error.message, true);
  }
}

// Сохранение организации
async function handleOrgSubmit(event) {
  event.preventDefault();
  
  const id = document.getElementById('orgId').value;
  const data = {
    name: document.getElementById('orgName').value,
    address: document.getElementById('orgAddress').value,
    phone: document.getElementById('orgPhone').value,
    email: document.getElementById('orgEmail').value
  };
  
  try {
    if (id) {
      await api.updateOrganization(id, data);
      showToast('Организация обновлена');
    } else {
      await api.createOrganization(data);
      showToast('Организация создана');
    }
    closeModal();
    loadOrganizations(currentOffset);
  } catch (error) {
    showToast('Ошибка сохранения: ' + error.message, true);
  }
}

// Удаление организации
async function deleteOrganization(id) {
  if (!await showConfirm('Вы уверены, что хотите удалить эту организацию?')) return;
  
  try {
    await api.deleteOrganization(id);
    showToast('Организация удалена');
    loadOrganizations(currentOffset);
  } catch (error) {
    showToast('Ошибка удаления: ' + error.message, true);
  }
}

// === УЧАСТНИКИ ОРГАНИЗАЦИИ ===

function openUsersModal(orgId, orgName) {
  currentOrgId = orgId;
  document.getElementById('usersModalTitle').textContent = `Участники: ${orgName}`;
  document.getElementById('currentOrgName').textContent = orgName;
  document.getElementById('usersModal').classList.add('active');
  loadOrganizationUsers(orgId);
}

function closeUsersModal() {
  document.getElementById('usersModal').classList.remove('active');
  currentOrgId = null;
}

async function loadOrganizationUsers(orgId) {
  try {
    const data = await api.getOrganizationUsers(orgId, 50, 0);
    renderOrganizationUsers(data.organization_users || []);
  } catch (error) {
    document.getElementById('usersTableBody').innerHTML = `
      <tr><td colspan="4" class="text-center text-muted">Ошибка: ${error.message}</td></tr>
    `;
  }
}

function renderOrganizationUsers(users) {
  const tbody = document.getElementById('usersTableBody');
  
  if (!users || users.length === 0) {
    tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">Участников нет</td></tr>';
    return;
  }
  
  const roleNames = {
    'admin': '👑 Администратор',
    'manager': '📋 Менеджер',
    'lab_technician': '🧪 Лаборант',
    'viewer': '👁️ Наблюдатель'
  };
  
  tbody.innerHTML = users.map(user => `
    <tr>
      <td><code>${user.user_id.substring(0, 8)}...</code></td>
      <td><span class="badge badge-primary">${roleNames[user.role] || user.role}</span></td>
      <td>${formatDate(user.created_at)}</td>
      <td>
        <div class="actions">
          <select onchange="updateUserRole('${user.id}', this.value)" style="padding: 6px 10px; border-radius: var(--radius); border: 1px solid var(--border);">
            <option value="">Сменить роль</option>
            <option value="admin">Администратор</option>
            <option value="manager">Менеджер</option>
            <option value="lab_technician">Лаборант</option>
            <option value="viewer">Наблюдатель</option>
          </select>
          <button class="btn btn-danger btn-small" onclick="deleteOrganizationUser('${user.id}')">🗑️</button>
        </div>
      </td>
    </tr>
  `).join('');
}

function openAddUserModal() {
  document.getElementById('addUserForm').reset();
  document.getElementById('addUserModal').classList.add('active');
}

function closeAddUserModal() {
  document.getElementById('addUserModal').classList.remove('active');
}

async function handleAddUserSubmit(event) {
  event.preventDefault();
  
  const userId = document.getElementById('newUserId').value;
  const role = document.getElementById('newUserRole').value;
  
  try {
    await api.createOrganizationUser(currentOrgId, {
      organization_id: currentOrgId,
      user_id: userId,
      role: role
    });
    showToast('Участник добавлен');
    closeAddUserModal();
    loadOrganizationUsers(currentOrgId);
  } catch (error) {
    showToast('Ошибка добавления участника: ' + error.message, true);
  }
}

async function updateUserRole(userId, newRole) {
  if (!newRole) return;
  
  try {
    await api.updateOrganizationUser(currentOrgId, userId, newRole);
    showToast('Роль обновлена');
    loadOrganizationUsers(currentOrgId);
  } catch (error) {
    showToast('Ошибка обновления роли: ' + error.message, true);
  }
}

async function deleteOrganizationUser(userId) {
  if (!await showConfirm('Удалить участника из организации?')) return;
  
  try {
    await api.deleteOrganizationUser(currentOrgId, userId);
    showToast('Участник удалён');
    loadOrganizationUsers(currentOrgId);
  } catch (error) {
    showToast('Ошибка удаления: ' + error.message, true);
  }
}

// === УСЛУГИ ОРГАНИЗАЦИИ ===

function openTestsModal(orgId, orgName) {
  currentOrgId = orgId;
  document.getElementById('testsModalTitle').textContent = `Услуги: ${orgName}`;
  document.getElementById('testsOrgName').textContent = orgName;
  document.getElementById('testsModal').classList.add('active');
  loadOrganizationTests(orgId);
}

function closeTestsModal() {
  document.getElementById('testsModal').classList.remove('active');
  currentOrgId = null;
}

async function loadOrganizationTests(orgId) {
  try {
    const data = await api.getOrganizationTests(50, 0);
    // Фильтруем тесты по организации (в API пока нет фильтрации по orgId для списка)
    const orgTests = (data.organization_tests || []).filter(t => t.organization_id === orgId);
    renderOrganizationTests(orgTests);
  } catch (error) {
    document.getElementById('testsTableBody').innerHTML = `
      <tr><td colspan="4" class="text-center text-muted">Ошибка: ${error.message}</td></tr>
    `;
  }
}

function renderOrganizationTests(tests) {
  const tbody = document.getElementById('testsTableBody');
  
  if (!tests || tests.length === 0) {
    tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">Услуг нет</td></tr>';
    return;
  }
  
  tbody.innerHTML = tests.map(test => `
    <tr>
      <td><code>${test.test_method_id.substring(0, 8)}...</code></td>
      <td><strong>${test.price.toFixed(2)} ₽</strong></td>
      <td>${escapeHtml(test.description) || '—'}</td>
      <td>
        <div class="actions">
          <button class="btn btn-warning btn-small" onclick="editTest('${test.id}', '${test.price}', '${escapeHtml(test.description)}')">✏️</button>
          <button class="btn btn-danger btn-small" onclick="deleteTest('${test.id}')">🗑️</button>
        </div>
      </td>
    </tr>
  `).join('');
}

function openAddTestModal() {
  document.getElementById('addTestForm').reset();
  document.getElementById('addTestModal').classList.add('active');
}

function closeAddTestModal() {
  document.getElementById('addTestModal').classList.remove('active');
}

async function handleAddTestSubmit(event) {
  event.preventDefault();
  
  const testMethodId = document.getElementById('testMethodId').value;
  const price = parseFloat(document.getElementById('testPrice').value);
  const description = document.getElementById('testDescription').value;
  
  try {
    await api.createOrganizationTest(currentOrgId, {
      organization_id: currentOrgId,
      test_method_id: testMethodId,
      price: price,
      description: description
    });
    showToast('Услуга добавлена');
    closeAddTestModal();
    loadOrganizationTests(currentOrgId);
  } catch (error) {
    showToast('Ошибка добавления услуги: ' + error.message, true);
  }
}

async function editTest(testId, price, description) {
  const newPrice = prompt('Новая цена:', price);
  if (newPrice === null) return;
  
  const newDescription = prompt('Новое описание:', description);
  if (newDescription === null) return;
  
  try {
    await api.updateOrganizationTest(currentOrgId, testId, {
      price: parseFloat(newPrice),
      description: newDescription
    });
    showToast('Услуга обновлена');
    loadOrganizationTests(currentOrgId);
  } catch (error) {
    showToast('Ошибка обновления: ' + error.message, true);
  }
}

async function deleteTest(testId) {
  if (!await showConfirm('Удалить эту услугу?')) return;
  
  try {
    await api.deleteOrganizationTest(currentOrgId, testId);
    showToast('Услуга удалена');
    loadOrganizationTests(currentOrgId);
  } catch (error) {
    showToast('Ошибка удаления: ' + error.message, true);
  }
}

// Утилита для экранирования HTML
function escapeHtml(text) {
  if (!text) return '';
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Экспорт функций для глобального доступа
window.openCreateModal = openCreateModal;
window.closeModal = closeModal;
window.handleOrgSubmit = handleOrgSubmit;
window.editOrganization = editOrganization;
window.deleteOrganization = deleteOrganization;
window.handleSearch = handleSearch;
window.loadOrganizations = loadOrganizations;
window.openUsersModal = openUsersModal;
window.closeUsersModal = closeUsersModal;
window.openAddUserModal = openAddUserModal;
window.closeAddUserModal = closeAddUserModal;
window.handleAddUserSubmit = handleAddUserSubmit;
window.updateUserRole = updateUserRole;
window.deleteOrganizationUser = deleteOrganizationUser;
window.openTestsModal = openTestsModal;
window.closeTestsModal = closeTestsModal;
window.openAddTestModal = openAddTestModal;
window.closeAddTestModal = closeAddTestModal;
window.handleAddTestSubmit = handleAddTestSubmit;
window.editTest = editTest;
window.deleteTest = deleteTest;
