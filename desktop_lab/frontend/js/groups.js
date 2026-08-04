// frontend/js/groups.js
import { api, downloadBlob } from './api.js';
import { formatDate, showToast, showConfirm } from './utils.js';
import { initNavigation } from './navigation.js';

let materials = [];
let currentPage = 1;
const limit = 20;

document.addEventListener('DOMContentLoaded', async () => {
  initNavigation();
  await loadMaterials();
  await loadGroups();
  setupGroupFilters();
});

async function loadMaterials() {
  try {
    materials = await api.getMaterials();
    const sel = document.getElementById('newGroupMaterial');
    if (!sel) return;
    
    sel.innerHTML = '<option value="">-- Выберите материал --</option>';
    materials.forEach(m => {
      sel.innerHTML += `<option value="${m.id}">${m.name}</option>`;
    });
    
    // УДАЛЕНО: попытка заполнить текстовый input тегами <option>
  } catch(e) {
    console.error('Load materials error:', e);
  }
}

// === FILTERS & PAGINATION ===
function setupGroupFilters() {
  const searchInput = document.getElementById('groupSearchInput');
  const filterProject = document.getElementById('filterProject');
  const filterMaterial = document.getElementById('filterMaterial');
  const prevPage = document.getElementById('prevPage');
  const nextPage = document.getElementById('nextPage');

  if (searchInput) searchInput.addEventListener('input', debounce(() => {
    currentPage = 1;
    loadGroups();
  }, 300));

  if (filterProject) filterProject.addEventListener('input', debounce(() => {
    currentPage = 1;
    loadGroups();
  }, 300));

  // ИСПРАВЛЕНО: используем 'input' + debounce для текстового поля, а не 'change'
  if (filterMaterial) filterMaterial.addEventListener('input', debounce(() => {
    currentPage = 1;
    loadGroups();
  }, 300));

  if (prevPage) prevPage.addEventListener('click', () => {
    if (currentPage > 1) { currentPage--; loadGroups(); }
  });

  if (nextPage) nextPage.addEventListener('click', () => {
    currentPage++;
    loadGroups();
  });
}

function debounce(fn, delay) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn.apply(this, args), delay);
  };
}

async function loadGroups() {
  const offset = (currentPage - 1) * limit;
  const table = document.getElementById('groupsTable');
  if (!table) return;
  table.innerHTML = '<tr><td colspan="6" class="text-center">Загрузка...</td></tr>';
  try {
    const searchQuery = document.getElementById('groupSearchInput')?.value || '';
    const projectQuery = document.getElementById('filterProject')?.value || '';
    const materialQuery = document.getElementById('filterMaterial')?.value || '';

    // Формируем query-параметры для фильтрации
    const params = new URLSearchParams();
    params.set('limit', limit);
    params.set('offset', offset);

    if (searchQuery) params.set('name', searchQuery);
    if (projectQuery) params.set('project_name', projectQuery);
    if (materialQuery) params.set('material', materialQuery);

    const res = await api.getGroups(params.toString());
    renderGroupsTable(res.items || [], res.meta);
  } catch(e) {
    console.error('Load groups error:', e);
    table.innerHTML = '<tr><td colspan="6" class="text-center text-danger">Ошибка загрузки</td></tr>';
  }
}

function renderGroupsTable(groups, meta) {
  const table = document.getElementById('groupsTable');
  if (!table) return;

  updatePagination(meta);

  if (!groups || !groups.length) {
    table.innerHTML = '<tr><td colspan="6" class="text-center text-muted">Нет групп</td></tr>';
    return;
  }

  let html = '';
  groups.forEach(g => {
    const mat = materials.find(m => m.id === g.material_id);
    html += `
      <tr>
        <td><strong>${g.name}</strong></td>
        <td>${g.project_name || '—'}</td>
        <td>${mat?.name || '—'}</td>
        <td class="text-muted">—</td>
        <td>${formatDate(g.created_at)}</td>
        <td>
          <div class="actions">
            <button class="btn btn-small btn-secondary" onclick="viewGroup('${g.id}')" title="Просмотр">👁️</button>
            <button class="btn btn-small btn-secondary" onclick="editGroup('${g.id}')" title="Редактировать">✏️</button>
            <button class="btn btn-small btn-secondary" onclick="downloadGroupPDF('${g.id}')" title="Скачать PDF">📥</button>
            <button class="btn btn-small btn-danger" onclick="deleteGroup('${g.id}')" title="Удалить">🗑️</button>
          </div>
        </td>
      </tr>`;
  });
  table.innerHTML = html;
}


function updatePagination(meta) {
  const pageInfo = document.getElementById('pageInfo');
  const prevPage = document.getElementById('prevPage');
  const nextPage = document.getElementById('nextPage');

  if (pageInfo) pageInfo.textContent = `Страница ${meta?.page || 1} из ${meta?.total_pages || 1}`;
  if (prevPage) prevPage.disabled = !meta?.has_prev_page;
  if (nextPage) nextPage.disabled = !meta?.has_next_page;
}

// === MODALS ===
function openCreateGroupModal() {
  document.getElementById('createGroupModal').classList.add('active');
}

function closeCreateGroupModal() {
  document.getElementById('createGroupModal').classList.remove('active');
  document.getElementById('newGroupName').value = '';
  document.getElementById('newGroupProject').value = '';
  document.getElementById('newGroupLocation').value = '';
  const sel = document.getElementById('newGroupMaterial');
  if (sel) sel.value = '';
}

async function createGroup() {
  const name = document.getElementById('newGroupName').value.trim();
  const materialId = document.getElementById('newGroupMaterial').value;
  if (!name || !materialId) {
    showToast('Заполните название и материал', true);
    return;
  }
  try {
    await api.createGroup({
      name,
      project_name: document.getElementById('newGroupProject').value,
      location: document.getElementById('newGroupLocation').value,
      material_id: materialId
    });
    showToast('Группа создана');
    closeCreateGroupModal();
    loadGroups();
  } catch(e) {
    console.error('Create group error:', e);
  }
}

// === VIEW GROUP & PROTOCOLS ===
async function viewGroup(id) {
  try {
    const group = await api.getGroupById(id);
    const mat = materials.find(m => m.id === group.material_id);

    document.getElementById('viewGroupName').textContent = group.name;
    document.getElementById('viewGroupInfo').innerHTML = `
      <div class="info-item"><span class="info-label">Проект</span><span class="info-value">${group.project_name || '—'}</span></div>
      <div class="info-item"><span class="info-label">Место</span><span class="info-value">${group.location || '—'}</span></div>
      <div class="info-item"><span class="info-label">Материал</span><span class="info-value">${mat?.name || '—'}</span></div>
      <div class="info-item"><span class="info-label">Создана</span><span class="info-value">${formatDate(group.created_at)}</span></div>
    `;

    // ✅ Загружаем протоколы ТОЛЬКО этой группы
    const protocols = await api.request(`/protocol/groups/${id}`);
    const protDiv = document.getElementById('viewGroupProtocols');

    if (!protocols || protocols.length === 0) {
      protDiv.innerHTML = '<p class="text-muted">Нет протоколов</p>';
    } else {
      let html = `<table>
        <thead><tr><th>№</th><th>Дата</th><th>Статус</th><th>Действия</th></tr></thead>
        <tbody>`;
      protocols.slice(0, 10).forEach(p => {
        html += `<tr>
          <td>${p.protocol_number || p.id.substring(0,8)}</td>
          <td>${formatDate(p.created_at)}</td>
          <td><span class="badge ${p.status==='draft'?'badge-warning':'badge-success'}">${p.status==='draft'?'Черновик':'Завершён'}</span></td>
          <td>
            <button class="btn btn-small btn-secondary" onclick="viewProtocol('${p.id}')" title="Просмотр">👁️</button>
          </td>
        </tr>`;
      });
      html += '</tbody></table>';
      if (protocols.length > 10) {
        html += `<p class="text-muted mt-2">+ ещё ${protocols.length - 10} протоколов</p>`;
      }
      protDiv.innerHTML = html;
    }

    document.getElementById('viewGroupModal').classList.add('active');
    document.getElementById('groupDownloadBtn').onclick = () => downloadGroupPDF(id);
  } catch(e) {
    console.error('viewGroup error:', e);
    showToast('Ошибка загрузки группы', true);
  }
}

function closeViewGroupModal() {
  document.getElementById('viewGroupModal').classList.remove('active');
}

// === EDIT GROUP ===

// Открытие модального окна редактирования с загрузкой данных
// === EDIT GROUP (без изменения материала) ===

async function editGroup(id) {
  try {
    // Загружаем текущие данные группы
    const group = await api.getGroupById(id);
    
    // Заполняем форму
    document.getElementById('editGroupId').value = id;
    document.getElementById('editGroupName').value = group.name || '';
    document.getElementById('editGroupProject').value = group.project_name || '';
    document.getElementById('editGroupLocation').value = group.location || '';
    
    // Показываем модальное окно
    document.getElementById('editGroupModal').classList.add('active');
  } catch(e) {
    console.error('editGroup error:', e);
    showToast('Ошибка загрузки данных группы', true);
  }
}

function closeEditGroupModal() {
  document.getElementById('editGroupModal').classList.remove('active');
  document.getElementById('editGroupId').value = '';
}

async function saveGroupUpdate() {
  const id = document.getElementById('editGroupId').value;
  const name = document.getElementById('editGroupName').value.trim();
  
  // Валидация: обязательно только название
  if (!id || !name) {
    showToast('Заполните название группы', true);
    return;
  }
  
  // Отправляем только разрешённые поля (без material_id)
  const payload = {
    name: name,
    project_name: document.getElementById('editGroupProject').value.trim(),
    location: document.getElementById('editGroupLocation').value.trim()
  };
  
  try {
    await api.updateGroup(id, payload);
    showToast('Группа обновлена');
    closeEditGroupModal();
    loadGroups(); // Перезагружаем таблицу
  } catch(e) {
    console.error('saveGroupUpdate error:', e);
    showToast('Ошибка сохранения: ' + e.message, true);
  }
}

// Экспорт для HTML onclick
window.editGroup = editGroup;
window.closeEditGroupModal = closeEditGroupModal;
window.saveGroupUpdate = saveGroupUpdate;

window.closeViewGroupModal = closeViewGroupModal;

// === DOWNLOAD PDF ===
async function downloadGroupPDF(id) {
  try {
    const blob = await api.downloadGroupPDF(id);
    downloadBlob(blob, `group_${id}_summary.pdf`);
    showToast('PDF отчёт скачан');
  } catch(e) {
    console.error('downloadGroupPDF error:', e);
    showToast('Ошибка генерации PDF: ' + e.message, true);
  }
}

// === DELETE GROUP ===
async function deleteGroup(id) {
  if (!await showConfirm('Удалить группу? Привязка протоколов сбросится.')) return;
  try {
    // ✅ Реальный DELETE запрос к API
    await api.request(`/group/${id}`, { method: 'DELETE' });
    showToast('Группа удалена');
    loadGroups();
  } catch(e) {
    console.error('deleteGroup error:', e);
    showToast('Ошибка удаления', true);
  }
}

// Make functions available globally
window.openCreateGroupModal = openCreateGroupModal;
window.closeCreateGroupModal = closeCreateGroupModal;
window.createGroup = createGroup;
window.viewGroup = viewGroup;
window.closeViewGroupModal = closeViewGroupModal;
window.downloadGroupPDF = downloadGroupPDF;
window.deleteGroup = deleteGroup;
window.editGroup = editGroup;
window.closeEditGroupModal = closeEditGroupModal;
window.saveGroupUpdate = saveGroupUpdate;

export { openCreateGroupModal, closeCreateGroupModal, createGroup, viewGroup, closeViewGroupModal, downloadGroupPDF, deleteGroup, editGroup, closeEditGroupModal, saveGroupUpdate };