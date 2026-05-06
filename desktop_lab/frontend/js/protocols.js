// frontend/js/protocols.js
import { api, downloadBlob } from './api.js';
import { formatDate, showToast, showConfirm, setLoading } from './utils.js';
import { initNavigation } from './navigation.js';

let currentPage = 1;
const limit = 20;
let viewingProtocolId = null;

document.addEventListener('DOMContentLoaded', async () => {
  initNavigation();
  await loadProtocols();
  setupFilters();
});

function setupFilters() {
  document.getElementById('searchInput').addEventListener('input', debounce(() => {
    currentPage = 1;
    loadProtocols();
  }, 300));
  
  document.getElementById('filterStatus').addEventListener('change', () => {
    currentPage = 1;
    loadProtocols();
  });
  
  document.getElementById('prevPage').addEventListener('click', () => {
    if (currentPage > 1) {
      currentPage--;
      loadProtocols();
    }
  });
  
  document.getElementById('nextPage').addEventListener('click', () => {
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
  table.innerHTML = '<tr><td colspan="6" class="text-center">Загрузка...</td></tr>';
  
  try {
    const res = await api.getProtocols(limit, offset);
    renderProtocolsTable(res.items || [], res.meta);
  } catch(e) {
    table.innerHTML = '<tr><td colspan="6" class="text-center text-danger">Ошибка загрузки</td></tr>';
  }
}

function renderProtocolsTable(protocols, meta) {
  const table = document.getElementById('protocolsTable');
  
  if (!protocols.length) {
    table.innerHTML = '<tr><td colspan="6" class="text-center text-muted">Нет протоколов</td></tr>';
    updatePagination(meta);
    return;
  }
  
  let html = '';
  protocols.forEach(p => {
    const statusClass = p.status === 'draft' ? 'badge-warning' : 'badge-success';
    const statusText = p.status === 'draft' ? 'Черновик' : 'Завершён';
    
    html += `
      <tr>
        <td><strong>${p.protocol_number || p.id.substring(0,8)}</strong></td>
        <td>${formatDate(p.created_at)}</td>
        <td>${p.lab_name || '—'}</td>
        <td>${p.sample_number || '—'}</td>
        <td><span class="badge ${statusClass}">${statusText}</span></td>
        <td>
          <div class="actions">
            <button class="btn btn-small btn-secondary" onclick="viewProtocol('${p.id}')">👁️</button>
            <button class="btn btn-small btn-secondary" onclick="downloadPDF('${p.id}')">📥</button>
            ${p.status === 'draft' ? 
              `<button class="btn btn-small btn-success" onclick="completeProtocol('${p.id}')">✓</button>` : ''}
            <button class="btn btn-small btn-danger" onclick="deleteProtocol('${p.id}')">🗑️</button>
          </div>
        </td>
      </tr>`;
  });
  
  table.innerHTML = html;
  updatePagination(meta);
}

function updatePagination(meta) {
  document.getElementById('pageInfo').textContent = `Страница ${meta?.page || 1} из ${meta?.total_pages || 1}`;
  document.getElementById('prevPage').disabled = !meta?.has_prev_page;
  document.getElementById('nextPage').disabled = !meta?.has_next_page;
}

// === ACTIONS ===
async function viewProtocol(id) {
  viewingProtocolId = id;
  
  try {
    const full = await api.getProtocolFull(id);
    
    // Info
    document.getElementById('viewInfo').innerHTML = `
      <div class="info-item"><span class="info-label">№ Протокола</span><span class="info-value">${full.protocol.protocol_number || '—'}</span></div>
      <div class="info-item"><span class="info-label">Дата</span><span class="info-value">${formatDate(full.protocol.test_date || full.protocol.created_at)}</span></div>
      <div class="info-item"><span class="info-label">Лаборатория</span><span class="info-value">${full.protocol.lab_name || '—'}</span></div>
      <div class="info-item"><span class="info-label">Оператор</span><span class="info-value">${full.protocol.operator_name || '—'}</span></div>
      <div class="info-item"><span class="info-label">Материал</span><span class="info-value">${full.material?.name || '—'}</span></div>
      <div class="info-item"><span class="info-label">Место отбора</span><span class="info-value">${full.sample?.collection_place || '—'}</span></div>
      <div class="info-item"><span class="info-label">Номер пробы</span><span class="info-value">${full.sample?.sample_number || '—'}</span></div>
    `;
    
    // Results
    const resultsDiv = document.getElementById('viewResults');
    if (!full.results?.length) {
      resultsDiv.innerHTML = '<p class="text-muted">Нет данных</p>';
    } else {
      let html = '<div class="table-wrap"><table><thead><tr><th>Метод</th><th>Значение</th><th>Норма</th><th>Статус</th></tr></thead><tbody>';
      
      for (const res of full.results) {
        let normStr = '—';
        // Упрощённо: берём первый лимит если есть
        if (res.applied_limit_id) {
          normStr = 'см. норматив';
        }
        
        const compliance = res.is_compliant === false 
          ? '<span class="badge badge-danger">✗</span>' 
          : '<span class="badge badge-success">✓</span>';
        
        html += `
          <tr>
            <td>${res.method_name || res.method_id.substring(0,20)}...</td>
            <td>${res.calculated_value != null ? res.calculated_value.toFixed(2) : '—'}</td>
            <td class="text-muted">${normStr}</td>
            <td>${compliance}</td>
          </tr>`;
      }
      html += '</tbody></table></div>';
      resultsDiv.innerHTML = html;
    }
    
    document.getElementById('viewModal').classList.add('active');
    document.getElementById('viewDownloadBtn').onclick = () => downloadPDF(id);
    
  } catch(e) {
    showToast('Ошибка загрузки протокола', true);
  }
}

window.closeViewModal = function() {
  document.getElementById('viewModal').classList.remove('active');
  viewingProtocolId = null;
};

async function downloadPDF(id) {
  try {
    const blob = await api.downloadProtocolPDF(id);
    downloadBlob(blob, `protocol_${id}.pdf`);
    showToast('PDF скачан');
  } catch(e) {}
}

async function completeProtocol(id) {
  if (!await showConfirm('Завершить протокол? Статус изменится на "Завершён".')) return;
  
  try {
    await api.updateProtocolStatus(id, 'completed');
    showToast('Протокол завершён');
    loadProtocols();
  } catch(e) {}
}

async function deleteProtocol(id) {
  if (!await showConfirm('Удалить протокол? Это действие нельзя отменить.')) return;
  
  try {
    await api.deleteProtocol(id);
    showToast('Протокол удалён');
    loadProtocols();
  } catch(e) {}
}

// Make functions available globally
window.viewProtocol = viewProtocol;
window.downloadPDF = downloadPDF;
window.completeProtocol = completeProtocol;
window.deleteProtocol = deleteProtocol;
window.closeViewModal = closeViewModal;