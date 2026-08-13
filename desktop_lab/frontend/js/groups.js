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
  table.innerHTML = '<tr><td colspan="8" class="text-center">Загрузка...</td></tr>';
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
    table.innerHTML = '<tr><td colspan="8" class="text-center text-danger">Ошибка загрузки</td></tr>';
  }
}

function renderGroupsTable(groups, meta) {
  const table = document.getElementById('groupsTable');
  if (!table) return;

  updatePagination(meta);

  if (!groups || !groups.length) {
    table.innerHTML = '<tr><td colspan="8" class="text-center text-muted">Нет групп</td></tr>';
    return;
  }

  let html = '';
  groups.forEach(g => {
    const mat = materials.find(m => m.id === g.material_id);
    const statusClass = g.status === 'draft' ? 'badge-warning' : (g.status === 'completed' ? 'badge-success' : 'badge-info');
    const statusLabel = g.status === 'draft' ? 'Черновик' : (g.status === 'completed' ? 'Завершено' : (g.status === 'in_progress' ? 'В работе' : g.status));
    html += `
      <tr>
        <td><strong>${g.name}</strong></td>
        <td>${g.project_name || '—'}</td>
        <td>${g.object_type || '—'}</td>
        <td>${mat?.name || '—'}</td>
        <td><span class="badge ${statusClass}">${statusLabel}</span></td>
        <td>${g.customer || '—'}</td>
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
export function openCreateGroupModal() {
  document.getElementById('createGroupModal').classList.add('active');
}

export function closeCreateGroupModal() {
  document.getElementById('createGroupModal').classList.remove('active');
  document.getElementById('newGroupName').value = '';
  document.getElementById('newGroupProject').value = '';
  document.getElementById('newGroupObjectType').value = '';
  document.getElementById('newGroupCustomer').value = '';
  document.getElementById('newGroupContractNumber').value = '';
  document.getElementById('newGroupStatus').value = 'draft';
  document.getElementById('newGroupResponsiblePerson').value = '';
  document.getElementById('newGroupLocation').value = '';
  const sel = document.getElementById('newGroupMaterial');
  if (sel) sel.value = '';
  document.getElementById('newGroupDescription').value = '';
}

// Делаем функции доступными глобально для onclick в HTML
window.openCreateGroupModal = openCreateGroupModal;
window.closeCreateGroupModal = closeCreateGroupModal;

async function createGroup() {
  const name = document.getElementById('newGroupName').value.trim();
  const projectName = document.getElementById('newGroupProject').value.trim();
  const materialId = document.getElementById('newGroupMaterial').value;
  
  if (!name || !projectName || !materialId) {
    showToast('Заполните название, проект и материал', true);
    return;
  }
  
  try {
    await api.createGroup({
      name,
      project_name: projectName,
      object_type: document.getElementById('newGroupObjectType').value.trim(),
      customer: document.getElementById('newGroupCustomer').value.trim(),
      contract_number: document.getElementById('newGroupContractNumber').value.trim(),
      status: document.getElementById('newGroupStatus').value,
      responsible_person_id: document.getElementById('newGroupResponsiblePerson').value.trim(),
      location: document.getElementById('newGroupLocation').value.trim(),
      material_id: materialId,
      description: document.getElementById('newGroupDescription').value.trim()
    });
    showToast('Группа создана');
    closeCreateGroupModal();
    loadGroups();
  } catch(e) {
    console.error('Create group error:', e);
  }
}

// === VIEW GROUP & PROTOCOLS ===
window.viewGroup = async function(id) {
  try {
    const group = await api.getGroupById(id);
    const mat = materials.find(m => m.id === group.material_id);

    document.getElementById('viewGroupName').textContent = group.name;
    
    const statusLabel = group.status === 'draft' ? 'Черновик' : (group.status === 'completed' ? 'Завершено' : (group.status === 'in_progress' ? 'В работе' : group.status));
    
    document.getElementById('viewGroupInfo').innerHTML = `
      <div class="info-item"><span class="info-label">Проект</span><span class="info-value">${group.project_name || '—'}</span></div>
      <div class="info-item"><span class="info-label">Тип объекта</span><span class="info-value">${group.object_type || '—'}</span></div>
      <div class="info-item"><span class="info-label">Заказчик</span><span class="info-value">${group.customer || '—'}</span></div>
      <div class="info-item"><span class="info-label">Контракт</span><span class="info-value">${group.contract_number || '—'}</span></div>
      <div class="info-item"><span class="info-label">Статус</span><span class="info-value"><span class="badge ${group.status === 'draft' ? 'badge-warning' : 'badge-success'}">${statusLabel}</span></span></div>
      <div class="info-item"><span class="info-label">Ответственный</span><span class="info-value">${group.responsible_person_id || '—'}</span></div>
      <div class="info-item"><span class="info-label">Место</span><span class="info-value">${group.location || '—'}</span></div>
      <div class="info-item"><span class="info-label">Материал</span><span class="info-value">${mat?.name || '—'}</span></div>
      <div class="info-item"><span class="info-label">Описание</span><span class="info-value">${group.description || '—'}</span></div>
      <div class="info-item"><span class="info-label">Создана</span><span class="info-value">${formatDate(group.created_at)}</span></div>
    `;

    // Загружаем протоколы группы
    const protocols = await api.getProtocolsByGroup(id);
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
    
    // Загружаем пробы группы
    try {
      const samples = await api.getSamplesByGroup(id);
      const sampDiv = document.getElementById('viewGroupSamples');
      
      if (!samples || samples.length === 0) {
        sampDiv.innerHTML = '<p class="text-muted">Нет проб</p>';
      } else {
        let html = `<table>
          <thead><tr><th>№</th><th>Место отбора</th><th>Дата</th><th>Действия</th></tr></thead>
          <tbody>`;
        samples.slice(0, 10).forEach(s => {
          html += `<tr>
            <td>${s.sample_number || s.id.substring(0,8)}</td>
            <td>${s.collection_place || '—'}</td>
            <td>${formatDate(s.collection_date)}</td>
            <td>
              <button class="btn btn-small btn-danger" onclick="removeSampleFromGroup('${id}', '${s.id}')" title="Удалить из группы">🗑️</button>
            </td>
          </tr>`;
        });
        html += '</tbody></table>';
        if (samples.length > 10) {
          html += `<p class="text-muted mt-2">+ ещё ${samples.length - 10} проб</p>`;
        }
        sampDiv.innerHTML = html;
      }
    } catch(e) {
      console.error('Error loading samples:', e);
      document.getElementById('viewGroupSamples').innerHTML = '<p class="text-muted">Ошибка загрузки проб</p>';
    }

    document.getElementById('viewGroupModal').classList.add('active');
    document.getElementById('groupDownloadBtn').onclick = () => downloadGroupPDF(id);
  } catch(e) {
    console.error('viewGroup error:', e);
    showToast('Ошибка загрузки группы', true);
  }
};

// === EDIT GROUP ===

// Открытие модального окна редактирования с загрузкой данных
export async function editGroup(id) {
  try {
    // Загружаем текущие данные группы
    const group = await api.getGroupById(id);
    
    // Заполняем форму
    document.getElementById('editGroupId').value = id;
    document.getElementById('editGroupName').value = group.name || '';
    document.getElementById('editGroupProject').value = group.project_name || '';
    document.getElementById('editGroupObjectType').value = group.object_type || '';
    document.getElementById('editGroupCustomer').value = group.customer || '';
    document.getElementById('editGroupContractNumber').value = group.contract_number || '';
    document.getElementById('editGroupStatus').value = group.status || 'draft';
    document.getElementById('editGroupResponsiblePerson').value = group.responsible_person_id || '';
    document.getElementById('editGroupLocation').value = group.location || '';
    document.getElementById('editGroupDescription').value = group.description || '';
    
    // Показываем модальное окно
    document.getElementById('editGroupModal').classList.add('active');
  } catch(e) {
    console.error('editGroup error:', e);
    showToast('Ошибка загрузки данных группы', true);
  }
}

export function closeEditGroupModal() {
  document.getElementById('editGroupModal').classList.remove('active');
  document.getElementById('editGroupId').value = '';
  document.getElementById('editGroupName').value = '';
  document.getElementById('editGroupProject').value = '';
  document.getElementById('editGroupObjectType').value = '';
  document.getElementById('editGroupCustomer').value = '';
  document.getElementById('editGroupContractNumber').value = '';
  document.getElementById('editGroupStatus').value = 'draft';
  document.getElementById('editGroupResponsiblePerson').value = '';
  document.getElementById('editGroupLocation').value = '';
  document.getElementById('editGroupDescription').value = '';
}

// Делаем функции доступными глобально для onclick в HTML
window.editGroup = editGroup;
window.closeEditGroupModal = closeEditGroupModal;

async function saveGroupUpdate() {
  const id = document.getElementById('editGroupId').value;
  const name = document.getElementById('editGroupName').value.trim();
  const projectName = document.getElementById('editGroupProject').value.trim();
  
  // Валидация: обязательно название и проект
  if (!id || !name || !projectName) {
    showToast('Заполните название группы и проект', true);
    return;
  }
  
  // Отправляем все поля для обновления
  const payload = {
    name: name,
    project_name: projectName,
    object_type: document.getElementById('editGroupObjectType').value.trim(),
    customer: document.getElementById('editGroupCustomer').value.trim(),
    contract_number: document.getElementById('editGroupContractNumber').value.trim(),
    status: document.getElementById('editGroupStatus').value,
    responsible_person_id: document.getElementById('editGroupResponsiblePerson').value.trim(),
    location: document.getElementById('editGroupLocation').value.trim(),
    description: document.getElementById('editGroupDescription').value.trim()
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

export function closeViewGroupModal() {
  document.getElementById('viewGroupModal').classList.remove('active');
}

// Делаем функцию доступной глобально для onclick в HTML
window.closeViewGroupModal = closeViewGroupModal;

// === DOWNLOAD PDF ===
export async function downloadGroupPDF(id) {
  try {
    const blob = await api.downloadGroupPDF(id);
    downloadBlob(blob, `group_${id}_summary.pdf`);
    showToast('PDF отчёт скачан');
  } catch(e) {
    console.error('downloadGroupPDF error:', e);
    showToast('Ошибка генерации PDF: ' + e.message, true);
  }
}

// Делаем функцию доступной глобально для onclick в HTML
window.downloadGroupPDF = downloadGroupPDF;

// === DELETE GROUP ===
export async function deleteGroup(id) {
  if (!await showConfirm('Удалить группу? Привязка протоколов сбросится.')) return;
  try {
    // Реальный DELETE запрос к API
    await api.request(`/group/${id}`, { method: 'DELETE' });
    showToast('Группа удалена');
    loadGroups();
  } catch(e) {
    console.error('deleteGroup error:', e);
    showToast('Ошибка удаления', true);
  }
}

// Делаем функцию доступной глобально для onclick в HTML
window.deleteGroup = deleteGroup;

// === REMOVE SAMPLE FROM GROUP ===
export async function removeSampleFromGroup(groupId, sampleId) {
  if (!await showConfirm('Удалить пробу из группы?')) return;
  try {
    await api.request(`/group/${groupId}/samples/${sampleId}`, { method: 'DELETE' });
    showToast('Проба удалена из группы');
    // Перезагружаем информацию о группе
    viewGroup(groupId);
  } catch(e) {
    console.error('removeSampleFromGroup error:', e);
    showToast('Ошибка удаления пробы', true);
  }
}

// Делаем функции доступными глобально
window.removeSampleFromGroup = removeSampleFromGroup;

// Make functions available globally
window.createGroup = createGroup;
window.viewGroup = viewGroup;
window.saveGroupUpdate = saveGroupUpdate;