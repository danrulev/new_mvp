// frontend/js/groups.js
import { api, downloadBlob } from './api.js';
import { formatDate, showToast, showConfirm } from './utils.js';
import { initNavigation } from './navigation.js';

let materials = [];

document.addEventListener('DOMContentLoaded', async () => {
  initNavigation();
  await loadMaterials();
  await loadGroups();
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
  } catch(e) {
    console.error('Load materials error:', e);
  }
}

async function loadGroups() {
  const table = document.getElementById('groupsTable');
  if (!table) return;
  table.innerHTML = '<tr><td colspan="6" class="text-center">Загрузка...</td></tr>';
  try {
    const res = await api.getGroups(50, 0);
    renderGroupsTable(res.items || []);
  } catch(e) {
    console.error('Load groups error:', e);
    table.innerHTML = '<tr><td colspan="6" class="text-center text-danger">Ошибка загрузки</td></tr>';
  }
}

function renderGroupsTable(groups) {
  const table = document.getElementById('groupsTable');
  if (!groups.length) {
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

// === MODALS ===
window.openCreateGroupModal = function() {
  document.getElementById('createGroupModal').classList.add('active');
};

window.closeCreateGroupModal = function() {
  document.getElementById('createGroupModal').classList.remove('active');
  document.getElementById('newGroupName').value = '';
  document.getElementById('newGroupProject').value = '';
  document.getElementById('newGroupLocation').value = '';
  const sel = document.getElementById('newGroupMaterial');
  if (sel) sel.value = '';
};

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
window.viewGroup = async function(id) {
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
        <thead><tr><th>№</th><th>Дата</th><th>Статус</th></tr></thead>
        <tbody>`;
      protocols.slice(0, 10).forEach(p => {
        html += `<tr>
          <td>${p.protocol_number || p.id.substring(0,8)}</td>
          <td>${formatDate(p.created_at)}</td>
          <td><span class="badge ${p.status==='draft'?'badge-warning':'badge-success'}">${p.status==='draft'?'Черновик':'Завершён'}</span></td>
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
};

// === EDIT GROUP ===

// Открытие модального окна редактирования с загрузкой данных
// === EDIT GROUP (без изменения материала) ===

window.editGroup = async function(id) {
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
};

window.closeEditGroupModal = function() {
  document.getElementById('editGroupModal').classList.remove('active');
  document.getElementById('editGroupId').value = '';
};

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

window.closeViewGroupModal = function() {
  document.getElementById('viewGroupModal').classList.remove('active');
};

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