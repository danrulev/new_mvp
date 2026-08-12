import { api } from './api.js';
import { showToast, setLoading } from './utils.js';
import { initNavigation } from './navigation.js';

// === STATE ===
let allSamples = [];
let materials = [];
let groups = [];

// === INIT ===
document.addEventListener('DOMContentLoaded', async () => {
  console.log('🚀 Samples page loaded');
  
  initNavigation();
  setupFormListeners();
});

function setupFormListeners() {
  const form = document.getElementById('sampleForm');
  if (form) {
    form.addEventListener('submit', handleFormSubmit);
  }
}

async function loadMaterials() {
  const sel = document.getElementById('materialSelect');
  const filterSel = document.getElementById('filterMaterial');
  if (!sel && !filterSel) return;

  try {
    materials = await api.getMaterials();
    
    // Заполняем селект в модалке
    if (sel) {
      sel.innerHTML = '<option value="">-- Выберите материал --</option>';
      materials.forEach(m => {
        const opt = document.createElement('option');
        opt.value = m.id;
        opt.textContent = m.name || 'Без названия';
        sel.appendChild(opt);
      });
    }
    
    // Заполняем фильтр
    if (filterSel) {
      filterSel.innerHTML = '<option value="">Все материалы</option>';
      materials.forEach(m => {
        const opt = document.createElement('option');
        opt.value = m.id;
        opt.textContent = m.name || 'Без названия';
        filterSel.appendChild(opt);
      });
    }
    
    console.log(`✅ Loaded ${materials.length} materials`);
  } catch (err) {
    console.error('❌ Failed to load materials:', err);
    showToast('Не удалось загрузить материалы', true);
  }
}

async function loadGroups() {
  const sel = document.getElementById('groupSelect');
  const filterSel = document.getElementById('filterGroup');
  if (!sel && !filterSel) return;

  try {
    const res = await api.getGroups('limit=100&offset=0');
    groups = res.items || (Array.isArray(res) ? res : []);
    
    // Заполняем селект в модалке
    if (sel) {
      sel.innerHTML = '<option value="">-- Без группы --</option>';
      groups.forEach(g => {
        const opt = document.createElement('option');
        opt.value = g.id;
        opt.textContent = g.name;
        sel.appendChild(opt);
      });
    }
    
    // Заполняем фильтр
    if (filterSel) {
      filterSel.innerHTML = '<option value="">Все группы</option>';
      groups.forEach(g => {
        const opt = document.createElement('option');
        opt.value = g.id;
        opt.textContent = g.name;
        filterSel.appendChild(opt);
      });
    }
    
    console.log(`✅ Loaded ${groups.length} groups`);
  } catch (err) {
    console.error('❌ Failed to load groups', err);
  }
}

async function loadSamples(groupFilter = '', materialFilter = '') {
  const container = document.getElementById('samplesContainer');
  if (!container) return;

  container.innerHTML = '<div class="text-muted">Загрузка образцов...</div>';

  try {
    let samples = [];
    
    if (groupFilter) {
      // Загружаем по группе
      samples = await api.getSamplesByGroup(groupFilter);
    } else {
      // Загружаем все образцы (через группы)
      // Для упрощения загружаем из первой группы или создаем пустой список
      // В реальном приложении нужен endpoint для получения всех образцов
      if (groups.length > 0) {
        for (const group of groups.slice(0, 10)) { // Ограничимся 10 группами
          const groupSamples = await api.getSamplesByGroup(group.id);
          samples = samples.concat(groupSamples || []);
        }
      }
    }
    
    // Фильтрация по материалу
    if (materialFilter) {
      samples = samples.filter(s => s.material_id === materialFilter);
    }
    
    allSamples = samples;
    
    if (!samples || samples.length === 0) {
      container.innerHTML = '<div class="text-muted">Нет доступных образцов</div>';
      return;
    }
    
    renderSamples(samples);
    console.log(`✅ Rendered ${samples.length} samples`);
  } catch (err) {
    console.error('❌ Failed to load samples:', err);
    container.innerHTML = '<div class="text-danger">Ошибка загрузки образцов</div>';
    showToast('Не удалось загрузить образцы', true);
  }
}

function renderSamples(samples) {
  const container = document.getElementById('samplesContainer');
  if (!container) return;

  let html = '';
  samples.forEach(sample => {
    const material = materials.find(m => m.id === sample.material_id);
    const group = groups.find(g => g.id === sample.group_id);
    
    html += `
      <div class="sample-card">
        <div class="sample-header">
          <div>
            <div class="sample-number">${escapeHtml(sample.sample_number)}</div>
            <div class="text-muted small">${material ? escapeHtml(material.name) : 'Неизвестный материал'}</div>
          </div>
          <div class="sample-actions">
            <button class="btn btn-sm btn-secondary" onclick="editSample('${sample.id}')">✏️</button>
            <button class="btn btn-sm btn-danger" onclick="deleteSample('${sample.id}')">🗑️</button>
          </div>
        </div>
        
        ${sample.photo_url ? `<img src="${escapeHtml(sample.photo_url)}" alt="Фото образца" class="sample-photo-preview mb-3">` : ''}
        
        <div class="sample-details-grid">
          ${sample.collection_place ? `
            <div class="detail-item">
              <div class="detail-label">Место отбора</div>
              <div class="detail-value">${escapeHtml(sample.collection_place)}</div>
            </div>
          ` : ''}
          
          ${sample.length_mm != null ? `
            <div class="detail-item">
              <div class="detail-label">Длина</div>
              <div class="detail-value">${sample.length_mm} мм</div>
            </div>
          ` : ''}
          
          ${sample.width_mm != null ? `
            <div class="detail-item">
              <div class="detail-label">Ширина</div>
              <div class="detail-value">${sample.width_mm} мм</div>
            </div>
          ` : ''}
          
          ${sample.height_mm != null ? `
            <div class="detail-item">
              <div class="detail-label">Высота</div>
              <div class="detail-value">${sample.height_mm} мм</div>
            </div>
          ` : ''}
          
          ${sample.weight_grams != null ? `
            <div class="detail-item">
              <div class="detail-label">Вес</div>
              <div class="detail-value">${sample.weight_grams} г</div>
            </div>
          ` : ''}
          
          ${sample.shape ? `
            <div class="detail-item">
              <div class="detail-label">Форма</div>
              <div class="detail-value">${translateShape(sample.shape)}</div>
            </div>
          ` : ''}
          
          ${sample.color ? `
            <div class="detail-item">
              <div class="detail-label">Цвет</div>
              <div class="detail-value">${escapeHtml(sample.color)}</div>
            </div>
          ` : ''}
          
          ${sample.batch_number ? `
            <div class="detail-item">
              <div class="detail-label">Партия</div>
              <div class="detail-value">${escapeHtml(sample.batch_number)}</div>
            </div>
          ` : ''}
          
          ${sample.manufacturer ? `
            <div class="detail-item">
              <div class="detail-label">Производитель</div>
              <div class="detail-value">${escapeHtml(sample.manufacturer)}</div>
            </div>
          ` : ''}
          
          ${group ? `
            <div class="detail-item">
              <div class="detail-label">Группа</div>
              <div class="detail-value">${escapeHtml(group.name)}</div>
            </div>
          ` : ''}
        </div>
        
        ${sample.note ? `
          <div class="mt-3 p-2" style="background: var(--bg-secondary); border-radius: var(--radius-md);">
            <small class="text-muted">${escapeHtml(sample.note)}</small>
          </div>
        ` : ''}
      </div>
    `;
  });
  
  container.innerHTML = html;
}

function translateShape(shape) {
  const shapes = {
    'cube': 'Куб',
    'cylinder': 'Цилиндр',
    'prism': 'Призма',
    'sphere': 'Сфера',
    'irregular': 'Неправильная'
  };
  return shapes[shape] || shape;
}

function escapeHtml(text) {
  if (!text) return '';
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// === MODAL FUNCTIONS ===

window.openCreateModal = function() {
  document.getElementById('modalTitle').textContent = 'Новый образец';
  document.getElementById('sampleForm').reset();
  document.getElementById('sampleId').value = '';
  document.getElementById('sampleModal').classList.add('active');
};

window.closeModal = function() {
  document.getElementById('sampleModal').classList.remove('active');
};

window.editSample = async function(id) {
  try {
    const sample = await api.getSampleById(id);
    if (!sample || !sample.id) {
      showToast('Образец не найден', true);
      return;
    }
    
    document.getElementById('modalTitle').textContent = 'Редактирование образца';
    document.getElementById('sampleId').value = sample.id;
    document.getElementById('sampleNumber').value = sample.sample_number || '';
    document.getElementById('materialSelect').value = sample.material_id || '';
    document.getElementById('groupSelect').value = sample.group_id || '';
    document.getElementById('collectionPlace').value = sample.collection_place || '';
    
    if (sample.collection_date) {
      const date = new Date(sample.collection_date);
      document.getElementById('collectionDate').value = date.toISOString().split('T')[0];
    }
    
    document.getElementById('lengthMM').value = sample.length_mm || '';
    document.getElementById('widthMM').value = sample.width_mm || '';
    document.getElementById('heightMM').value = sample.height_mm || '';
    document.getElementById('weightGrams').value = sample.weight_grams || '';
    document.getElementById('shape').value = sample.shape || '';
    document.getElementById('color').value = sample.color || '';
    document.getElementById('batchNumber').value = sample.batch_number || '';
    document.getElementById('manufacturer').value = sample.manufacturer || '';
    document.getElementById('photoURL').value = sample.photo_url || '';
    document.getElementById('note').value = sample.note || '';
    
    document.getElementById('sampleModal').classList.add('active');
  } catch (err) {
    console.error('Failed to load sample:', err);
    showToast('Не удалось загрузить данные образца', true);
  }
};

async function handleFormSubmit(e) {
  e.preventDefault();
  
  const sampleId = document.getElementById('sampleId').value;
  const sampleNumber = document.getElementById('sampleNumber').value.trim();
  const materialId = document.getElementById('materialSelect').value;
  const groupId = document.getElementById('groupSelect').value;
  
  if (!sampleNumber || !materialId) {
    showToast('Заполните обязательные поля', true);
    return;
  }
  
  // Собираем данные формы
  const collectionDateVal = document.getElementById('collectionDate').value;
  const collectionDate = collectionDateVal ? new Date(collectionDateVal + 'T00:00:00Z').toISOString() : null;
  
  const lengthMM = document.getElementById('lengthMM').value ? parseFloat(document.getElementById('lengthMM').value) : null;
  const widthMM = document.getElementById('widthMM').value ? parseFloat(document.getElementById('widthMM').value) : null;
  const heightMM = document.getElementById('heightMM').value ? parseFloat(document.getElementById('heightMM').value) : null;
  const weightGrams = document.getElementById('weightGrams').value ? parseFloat(document.getElementById('weightGrams').value) : null;
  
  const payload = {
    sample_number: sampleNumber,
    material_id: materialId,
    collection_place: document.getElementById('collectionPlace').value.trim(),
    collection_date: collectionDate,
    group_id: groupId || null,
    length_mm: lengthMM,
    width_mm: widthMM,
    height_mm: heightMM,
    weight_grams: weightGrams,
    shape: document.getElementById('shape').value,
    color: document.getElementById('color').value.trim(),
    batch_number: document.getElementById('batchNumber').value.trim(),
    manufacturer: document.getElementById('manufacturer').value.trim(),
    photo_url: document.getElementById('photoURL').value.trim(),
    note: document.getElementById('note').value.trim()
  };
  
  try {
    const btn = e.target.querySelector('button[type="submit"]');
    setLoading(btn, true);
    
    if (sampleId) {
      // Обновление
      await api.updateSample(sampleId, payload);
      showToast('Образец успешно обновлен');
    } else {
      // Создание
      // Для создания нужно указать group_id
      if (!groupId) {
        showToast('Выберите группу для образца', true);
        setLoading(btn, false);
        return;
      }
      await api.createSample({ ...payload, group_id: groupId });
      showToast('Образец успешно создан');
    }
    
    closeModal();
    await loadSamples();
  } catch (err) {
    console.error('Save error:', err);
    showToast('Ошибка сохранения: ' + (err.message || 'Неизвестная ошибка'), true);
  } finally {
    const btn = e.target.querySelector('button[type="submit"]');
    setLoading(btn, false);
  }
}

window.deleteSample = async function(id) {
  if (!confirm('Вы уверены, что хотите удалить этот образец?')) {
    return;
  }
  
  try {
    await api.deleteSample(id);
    showToast('Образец успешно удален');
    await loadSamples();
  } catch (err) {
    console.error('Delete error:', err);
    showToast('Ошибка удаления: ' + (err.message || 'Неизвестная ошибка'), true);
  }
};

window.applyFilters = function() {
  const groupFilter = document.getElementById('filterGroup').value;
  const materialFilter = document.getElementById('filterMaterial').value;
  loadSamples(groupFilter, materialFilter);
};

window.resetFilters = function() {
  document.getElementById('filterGroup').value = '';
  document.getElementById('filterMaterial').value = '';
  loadSamples();
};

// Экспорт функций для глобального доступа
window.loadMaterials = loadMaterials;
window.loadGroups = loadGroups;
window.loadSamples = loadSamples;
