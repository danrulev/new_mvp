// frontend/js/protocols.js
import { api, downloadBlob } from './api.js';
import { formatDate, formatDateOnly, showToast, showConfirm, setLoading } from './utils.js';
import { initNavigation } from './navigation.js';

let currentPage = 1;
const limit = 10;
let viewingProtocolId = null;
let editGroupsCache = [];

// === INIT ===
document.addEventListener('DOMContentLoaded', async () => {
  initNavigation();
  await loadEditGroups(); // Сначала загружаем группы для редактирования
  await loadProtocols();
  setupFilters();
});

// === FILTERS & PAGINATION ===
function setupFilters() {
  const searchProtocolId = document.getElementById('searchProtocolId');
  const searchLab = document.getElementById('searchLab');
  const filterStatus = document.getElementById('filterStatus');
  const filterStartDate = document.getElementById('filterStartDate');
  const filterEndDate = document.getElementById('filterEndDate');
  const prevPage = document.getElementById('prevPage');
  const nextPage = document.getElementById('nextPage');
  
  if (searchProtocolId) searchProtocolId.addEventListener('input', debounce(() => {
    currentPage = 1;
    loadProtocols();
  }, 300));
  
  if (filterStatus) filterStatus.addEventListener('change', () => {
    currentPage = 1;
    loadProtocols();
  });

  if (searchLab) searchLab.addEventListener('input', debounce(() => {
    currentPage = 1;
    loadProtocols();
  }, 300));
  
  if (filterStartDate) filterStartDate.addEventListener('change', () => {
    currentPage = 1;
    loadProtocols();
  });
  
  if (filterEndDate) filterEndDate.addEventListener('change', () => {
    currentPage = 1;
    loadProtocols();
  });
  
  if (prevPage) prevPage.addEventListener('click', () => {
    if (currentPage > 1) { currentPage--; loadProtocols(); }
  });
  
  if (nextPage) nextPage.addEventListener('click', () => {
    currentPage++;
    loadProtocols();
  });
}

function debounce(fn, delay) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn.apply(this, args), delay);
  };
}

async function loadProtocols() {
  const offset = (currentPage - 1) * limit;
  const table = document.getElementById('protocolsTable');
  if (!table) return;
  
  table.innerHTML = '<tr><td colspan="6" class="text-center">Загрузка...</td></tr>';
  
  try {
    const statusFilter = document.getElementById('filterStatus')?.value || '';
    const protocolIdQuery = document.getElementById('searchProtocolId')?.value || '';
    const labNameQuery = document.getElementById('searchLab')?.value || '';
    const startDate = document.getElementById('filterStartDate')?.value || '';
    const endDate = document.getElementById('filterEndDate')?.value || '';
    
    // Формируем query-параметры для фильтрации
    const params = new URLSearchParams();
    params.set('limit', limit);
    params.set('offset', offset);
    
    if (statusFilter) params.set('status', statusFilter);
    if (protocolIdQuery) params.set('protocol_id', protocolIdQuery);
    if (labNameQuery) params.set('lab_name', labNameQuery);
    if (startDate) params.set('start_test_date', startDate);
    if (endDate) params.set('end_test_date', endDate);
    
    const res = await api.getProtocols(params.toString());
    renderProtocolsTable(res.items || [], res.meta);
  } catch(e) {
    console.error('Load protocols error:', e);
    table.innerHTML = '<tr><td colspan="6" class="text-center text-danger">Ошибка загрузки</td></tr>';
  }
}

function renderProtocolsTable(protocols, meta) {
  const table = document.getElementById('protocolsTable');
  if (!table) return;
  
  if (!protocols || !protocols.length) {
    table.innerHTML = '<tr><td colspan="6" class="text-center text-muted">Нет протоколов</td></tr>';
    updatePagination(meta);
    return;
  }
  
  let html = '';
  protocols.forEach(p => {
    const isDraft = p.status === 'draft';
    const statusClass = isDraft ? 'badge-warning' : 'badge-success';
    const statusText = isDraft ? 'Черновик' : 'Завершён';
    
    html += `
      <tr>
        <td><strong>${p.protocol_number || (p.id ? p.id.substring(0,8) : '')}</strong></td>
        <td>${formatDateOnly(p.created_at)}</td>
        <td>${p.lab_name || '—'}</td>
        <td>${p.sample_number || '—'}</td>
        <td><span class="badge ${statusClass}">${statusText}</span></td>
        <td>
          <div class="actions">
            <button class="btn btn-small btn-secondary" onclick="viewProtocol('${p.id}')" title="Просмотр">👁️</button>
            <button class="btn btn-small btn-secondary" onclick="downloadPDF('${p.id}')" title="Скачать PDF">📥</button>
            ${isDraft ? 
              `<button class="btn btn-small btn-warning" onclick="editProtocol('${p.id}')" title="Редактировать">✏️</button>` : 
              `<button class="btn btn-small btn-secondary" disabled title="Только для черновиков">✏️</button>`}
            ${isDraft ? 
              `<button class="btn btn-small btn-success" onclick="completeProtocol('${p.id}')" title="Завершить">✓</button>` : ''}
            <button class="btn btn-small btn-danger" onclick="deleteProtocol('${p.id}')" title="Удалить">🗑️</button>
          </div>
        </td>
      </tr>`;
  });
  
  table.innerHTML = html;
  updatePagination(meta);
}

function updatePagination(meta) {
  const pageInfo = document.getElementById('pageInfo');
  const prevPage = document.getElementById('prevPage');
  const nextPage = document.getElementById('nextPage');
  
  if (pageInfo) pageInfo.textContent = `Страница ${meta?.page || 1} из ${meta?.total_pages || 1}`;
  if (prevPage) prevPage.disabled = !meta?.has_prev_page;
  if (nextPage) nextPage.disabled = !meta?.has_next_page;
}

// === VIEW PROTOCOL ===
async function viewProtocol(id) {
  viewingProtocolId = id;
  
  try {
    const full = await api.getProtocolFull(id);
    
    // Рендер заметок
    const notesContainer = document.getElementById('viewNotes');
    if (notesContainer) {
      const protocolNote = full.protocol?.note;
      const sampleNote = full.sample?.note;
      
      let notesHtml = '';
      if (protocolNote || sampleNote) {
        notesHtml = '<div class="notes-grid">';
        if (protocolNote) notesHtml += renderNoteBlock('Заметка к протоколу', protocolNote, 'note-protocol');
        if (sampleNote) notesHtml += renderNoteBlock('Заметка к пробе', sampleNote, 'note-sample');
        notesHtml += '</div>';
      }
      notesContainer.innerHTML = notesHtml;
    }
    
    // Основная информация
    const infoDiv = document.getElementById('viewInfo');
    if (infoDiv) {
      infoDiv.innerHTML = `
        <div class="info-item"><span class="info-label">№ Протокола</span><span class="info-value">${full.protocol?.protocol_number || '—'}</span></div>
        <div class="info-item"><span class="info-label">Дата испытания</span><span class="info-value">${formatDateOnly(full.protocol?.test_date || full.protocol?.created_at)}</span></div>
        <div class="info-item"><span class="info-label">Лаборатория</span><span class="info-value">${full.protocol?.lab_name || '—'}</span></div>
        <div class="info-item"><span class="info-label">Оператор</span><span class="info-value">${full.protocol?.operator_name || '—'}</span></div>
        <div class="info-item"><span class="info-label">Материал</span><span class="info-value">${full.material?.name || '—'}</span></div>
        <div class="info-item"><span class="info-label">Место отбора</span><span class="info-value">${full.sample?.collection_place || '—'}</span></div>
        <div class="info-item"><span class="info-label">Дата отбора</span><span class="info-value">${formatDateOnly(full.sample?.collection_date)}</span></div>
        <div class="info-item"><span class="info-label">Номер пробы</span><span class="info-value">${full.sample?.sample_number || '—'}</span></div>
      `;
    }
    
    // Результаты (без изменений)
    const resultsDiv = document.getElementById('viewResults');
    if (resultsDiv) {
      if (!full.results?.length) {
        resultsDiv.innerHTML = '<p class="text-muted">Нет данных</p>';
      } else {
        let html = '<div class="table-wrap"><table><thead><tr><th>Метод</th><th>Значение</th><th>Норма</th><th>Статус</th><th>Заметка</th></tr></thead><tbody>';
        
        for (const res of full.results) {
          let normStr = '—';
          const unit = res.method_unit || '';
          
          // 🔥 Формируем читаемый норматив на основе данных из БД
          if (res.limit_type) {
            if (res.limit_type === 'min' && res.min_value != null) {
              normStr = `≥ ${res.min_value} ${unit}`.trim();
            } else if (res.limit_type === 'max' && res.max_value != null) {
              normStr = `≤ ${res.max_value} ${unit}`.trim();
            } else if (res.limit_type === 'range' && res.min_value != null && res.max_value != null) {
              normStr = `${res.min_value} – ${res.max_value} ${unit}`.trim();
            } else {
              normStr = 'см. норматив'; // Фоллбэк, если тип лимита странный
            }
          }
          
          const compliance = res.is_compliant === false 
            ? '<span class="badge badge-danger" title="Не соответствует">✗</span>' 
            : '<span class="badge badge-success" title="Соответствует">✓</span>';
          
          const resultNote = res.note 
            ? `<span class="note-inline" title="${res.note.replace(/"/g, '&quot;')}">📝</span>` 
            : '<span class="text-muted">—</span>';
          
          html += `
            <tr>
              <td>${res.method_name || (res.method_id ? res.method_id.substring(0,20) + '...' : '—')}</td>
              <td>${res.calculated_value != null ? res.calculated_value.toFixed(2) : '—'}</td>
              <td class="text-muted">${normStr}</td>
              <td>${compliance}</td>
              <td>${resultNote}</td>
            </tr>`;
        }
        html += '</tbody></table></div>';
        html += '<div id="resultNotesDetail" class="mt-3"></div>';
        resultsDiv.innerHTML = html;
        
        document.querySelectorAll('.note-inline').forEach((icon, idx) => {
          icon.style.cursor = 'pointer';
          icon.onclick = () => {
            const note = full.results[idx].note;
            const method = full.results[idx].method_name || full.results[idx].method_id;
            const detailDiv = document.getElementById('resultNotesDetail');
            if (note && detailDiv) {
              detailDiv.innerHTML = `
                <div class="note-block note-result">
                  <span class="note-label">📝 Заметка: ${method}</span>
                  <div class="note-content">${note.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>')}</div>
                </div>`;
            }
          };
        });
      }
    }
    
    // Показать модальное окно
    const modal = document.getElementById('viewModal');
    if (modal) {
      modal.classList.add('active');
      const downloadBtn = document.getElementById('viewDownloadBtn');
      if (downloadBtn) downloadBtn.onclick = () => downloadPDF(id);
    }
    
  } catch(e) {
    console.error('View protocol error:', e);
    showToast('Ошибка загрузки протокола', true);
  }
}

// === EDIT PROTOCOL ===

async function loadEditGroups() {
  try {
     const params = new URLSearchParams();
    params.set('limit', '100');
    params.set('offset', '0');

    const res = await api.getGroups(params.toString());
    editGroupsCache = res.items || (Array.isArray(res) ? res : []);
  } catch(e) {
    console.error('Failed to load groups for edit modal', e);
    editGroupsCache = [];
  }
}

async function editProtocol(id) {
  try {
    const full = await api.getProtocolFull(id);
    const protocol = full.protocol;
    const sample = full.sample;
    
    document.getElementById('editProtocolId').value = id;
    document.getElementById('editLabName').value = protocol?.lab_name || '';
    document.getElementById('editOperatorName').value = protocol?.operator_name || '';
    document.getElementById('editSampleNumber').value = sample?.sample_number || '';
    document.getElementById('editCollectionPlace').value = sample?.collection_place || '';
    document.getElementById('editNote').value = protocol?.note || '';
    
    // Даты
    if (protocol?.test_date) {
      document.getElementById('editTestDate').value = new Date(protocol.test_date).toISOString().split('T')[0];
    } else {
      document.getElementById('editTestDate').value = '';
    }
    if (sample?.collection_date) {
      document.getElementById('editCollectionDate').value = new Date(sample.collection_date).toISOString().split('T')[0];
    } else {
      document.getElementById('editCollectionDate').value = '';
    }
    
    // Группа
    const groupSelect = document.getElementById('editGroupSelect');
    groupSelect.innerHTML = '<option value="">-- Без группы --</option>';
    editGroupsCache.forEach(g => {
      const opt = document.createElement('option');
      opt.value = g.id;
      opt.textContent = g.name;
      if (g.id === protocol?.group_id) opt.selected = true;
      groupSelect.appendChild(opt);
    });
    
    document.getElementById('editModal').classList.add('active');
    
  } catch(e) {
    console.error('Edit protocol error:', e);
    showToast('Ошибка загрузки данных для редактирования', true);
  }
}

function closeEditModal() {
  const modal = document.getElementById('editModal');
  if (modal) modal.classList.remove('active');
  const form = document.getElementById('editForm');
  if (form) form.reset();
  document.getElementById('editProtocolId').value = '';
}

async function saveProtocolEdit() {
  const id = document.getElementById('editProtocolId').value;
  if (!id) {
    showToast('Ошибка: нет ID протокола', true);
    return;
  }
  
  const payload = {
    group_id: document.getElementById('editGroupSelect').value || null,
    lab_name: document.getElementById('editLabName').value.trim(),
    operator_name: document.getElementById('editOperatorName').value.trim(),
    sample_number: document.getElementById('editSampleNumber').value.trim(),
    collection_place: document.getElementById('editCollectionPlace').value.trim(),
    note: document.getElementById('editNote').value.trim() || null
  };
  
  const testDate = document.getElementById('editTestDate').value;
  if (testDate) payload.test_date = new Date(testDate).toISOString();
  
  const collectionDate = document.getElementById('editCollectionDate').value;
  if (collectionDate) payload.collection_date = new Date(collectionDate).toISOString();
  
  if (!payload.lab_name || !payload.operator_name || !payload.sample_number || !payload.collection_place) {
    showToast('Заполните все обязательные поля', true);
    return;
  }
  
  try {
    const btn = document.getElementById('editSaveBtn');
    setLoading(btn, true);
    btn.textContent = 'Сохранение...';
    
    await api.updateProtocol(id, payload);
    
    showToast('Протокол обновлён');
    closeEditModal();
    loadProtocols();
    
  } catch(e) {
    console.error('Save edit error:', e);
  } finally {
    const btn = document.getElementById('editSaveBtn');
    setLoading(btn, false);
    btn.textContent = '💾 Сохранить изменения';
  }
}

// === HELPERS ===

function renderNoteBlock(label, note, className = '') {
  if (!note || typeof note !== 'string' || note.trim() === '') return '';
  
  const escapedNote = note
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\n/g, '<br>');
  
  return `
    <div class="note-block ${className}">
      <span class="note-label">📝 ${label}</span>
      <div class="note-content">${escapedNote}</div>
    </div>`;
}

async function downloadPDF(id) {
  try {
    const blob = await api.downloadProtocolPDF(id);
    downloadBlob(blob, `protocol_${id}.pdf`);
    showToast('PDF скачан');
  } catch(e) {
    console.error('Download PDF error:', e);
    showToast('Ошибка скачивания PDF', true);
  }
}

async function completeProtocol(id) {
  if (!await showConfirm('Завершить протокол? Статус изменится на "Завершён".')) return;
  try {
    await api.updateProtocolStatus(id, 'completed');
    showToast('Протокол завершён');
    loadProtocols();
  } catch(e) {
    console.error('Complete protocol error:', e);
    showToast('Ошибка завершения протокола', true);
  }
}

async function deleteProtocol(id) {
  if (!await showConfirm('Удалить протокол? Это действие нельзя отменить.')) return;
  try {
    await api.deleteProtocol(id);
    showToast('Протокол удалён');
    loadProtocols();
  } catch(e) {
    console.error('Delete protocol error:', e);
    showToast('Ошибка удаления протокола', true);
  }
}

function closeViewModal() {
  const modal = document.getElementById('viewModal');
  if (modal) modal.classList.remove('active');
  viewingProtocolId = null;
  document.getElementById('viewNotes').innerHTML = '';
  document.getElementById('viewInfo').innerHTML = '';
  document.getElementById('viewResults').innerHTML = '';
}

// === GLOBAL EXPORTS (ОБЯЗАТЕЛЬНО для type="module") ===
window.viewProtocol = viewProtocol;
window.downloadPDF = downloadPDF;
window.completeProtocol = completeProtocol;
window.deleteProtocol = deleteProtocol;
window.closeViewModal = closeViewModal;
window.editProtocol = editProtocol;
window.closeEditModal = closeEditModal;
window.saveProtocolEdit = saveProtocolEdit;


export { viewProtocol, downloadPDF, completeProtocol, deleteProtocol, closeViewModal, editProtocol, closeEditModal, saveProtocolEdit };