import { api } from './api.js';
import { showToast, setLoading } from './utils.js';
import { initNavigation } from './navigation.js';

// === STATE ===
let standards = [], methods = [];
let currentMethod = null;
let standardDimensions = []; // 🔥 Новое: храним измерения стандарта

// === INIT ===
document.addEventListener('DOMContentLoaded', async () => {
  console.log('🚀 DOM Loaded, initializing...');
  
  initNavigation();
  setupFormListeners();
  setupCascadingSelects();
  
  await loadMaterials();
  await loadGroups();
  await loadEmployees();
});

function setupFormListeners() {
  const optionalInputs = ['sampleNote', 'protocolNote', 'collectionDate', 'lengthMM', 'widthMM', 'heightMM', 'weightGrams', 'shape', 'color', 'batchNumber', 'manufacturer', 'photoURL'];
  optionalInputs.forEach(id => {
    const el = document.getElementById(id);
    if (el) {
      el.addEventListener('input', checkCanSave);
      el.addEventListener('change', checkCanSave);
    }
  });
  
  const requiredInputs = ['sampleNumber', 'samplePlace', 'labName', 'operator'];
  requiredInputs.forEach(id => {
    const el = document.getElementById(id);
    if (el) {
      el.addEventListener('input', checkCanSave);
      el.addEventListener('change', checkCanSave);
    }
  });
}

function setupCascadingSelects() {
  const matSel = document.getElementById('materialSelect');
  const stdSel = document.getElementById('standardSelect');
  const mthSel = document.getElementById('methodSelect');

  if (!matSel || !stdSel || !mthSel) {
    console.error('❌ Select elements not found!');
    return;
  }

  // Обработчик выбора материала
  matSel.addEventListener('change', async (e) => {
    const matId = e.target.value;
    
    stdSel.value = '';
    mthSel.value = '';
    mthSel.innerHTML = '<option value="">-- Выберите стандарт --</option>';
    mthSel.disabled = true;
    document.getElementById('methodDetails').innerHTML = '';
    currentMethod = null;
    
    // 🔥 Сбрасываем измерения
    standardDimensions = [];
    document.getElementById('dimensionsContainer').innerHTML = '<p class="text-muted">Выберите стандарт...</p>';
    document.getElementById('dimensionsCard').style.display = 'none';
    
    checkCanSave();

    if (!matId) {
      stdSel.innerHTML = '<option value="">-- Выберите материал --</option>';
      stdSel.disabled = true;
      return;
    }

    stdSel.disabled = true;
    stdSel.innerHTML = '<option value="">⏳ Загрузка стандартов...</option>';

    try {
      standards = await api.getStandardsByMaterial(matId);
      stdSel.innerHTML = '<option value="">-- Выберите стандарт --</option>';
      
      if (!standards || standards.length === 0) {
        stdSel.innerHTML += '<option value="" disabled>⚠️ Нет стандартов</option>';
        showToast('Для выбранного материала нет доступных стандартов', false);
      } else {
        standards.forEach(s => {
          const opt = document.createElement('option');
          opt.value = s.id;
          opt.textContent = s.name;
          stdSel.appendChild(opt);
        });
        stdSel.disabled = false;
      }
    } catch (err) {
      console.error('Error loading standards:', err);
      stdSel.innerHTML = '<option value="">❌ Ошибка загрузки</option>';
      showToast('Не удалось загрузить стандарты', true);
    }
  });

  // Обработчик выбора стандарта
  stdSel.addEventListener('change', async (e) => {
    const stdId = e.target.value;
    
    mthSel.value = '';
    document.getElementById('methodDetails').innerHTML = '';
    currentMethod = null;
    checkCanSave();

    if (!stdId) {
      mthSel.innerHTML = '<option value="">-- Выберите стандарт --</option>';
      mthSel.disabled = true;
      standardDimensions = [];
      document.getElementById('dimensionsCard').style.display = 'none';
      return;
    }

    mthSel.disabled = false;
    mthSel.innerHTML = '<option value="">-- Загрузка... --</option>';

    try {
      // 🔥 Параллельно загружаем методы и измерения стандарта
      const [methodsData, dimsData] = await Promise.all([
        api.getMethodsByStandard(stdId),
        api.getStandardDimensions(stdId)
      ]);
      
      methods = methodsData;
      mthSel.innerHTML = '<option value="">-- Выберите метод --</option>';
      
      if (!methods || methods.length === 0) {
        mthSel.innerHTML += '<option value="" disabled>⚠️ Нет методов</option>';
      } else {
        methods.forEach(m => {
          const opt = document.createElement('option');
          opt.value = m.id;
          opt.textContent = `${m.name}${m.unit ? ` (${m.unit})` : ''}`;
          mthSel.appendChild(opt);
        });
        mthSel.disabled = false;
      }
      
      // 🔥 Рендерим измерения
      standardDimensions = dimsData || [];
      renderDimensions(standardDimensions);
      
    } catch (err) {
      console.error('Error loading methods/dimensions:', err);
      mthSel.innerHTML = '<option value="">❌ Ошибка загрузки</option>';
      showToast('Не удалось загрузить методы или параметры', true);
    }
  });

  // Обработчик выбора метода
  mthSel.addEventListener('change', async (e) => {
    const methodId = e.target.value;
    const detailsContainer = document.getElementById('methodDetails');
    detailsContainer.innerHTML = '';
    currentMethod = null;
    checkCanSave();

    if (!methodId) return;

    try {
      const full = await api.getMethodDetails(methodId);
      currentMethod = { ...full.method, inputs: full.inputs, limits: full.limits };
      renderMethodDetails(full);
      checkCanSave();
    } catch (err) {
      console.error('Error loading method details:', err);
      detailsContainer.innerHTML = '<div class="text-danger">❌ Ошибка загрузки метода</div>';
      showToast('Не удалось загрузить детали метода', true);
    }
  });
}

// 🔥 НОВАЯ ФУНКЦИЯ: Рендеринг измерений стандарта
function renderDimensions(dims) {
  const container = document.getElementById('dimensionsContainer');
  const card = document.getElementById('dimensionsCard');
  
  if (!dims || dims.length === 0) {
    card.style.display = 'none';
    container.innerHTML = '';
    return;
  }
  
  // Добавляем класс анимации
  card.classList.remove('fade-in');
  void card.offsetWidth; // trigger reflow
  card.classList.add('fade-in');
  
  card.style.display = 'block';
  
  let html = '';
  dims.forEach((dim, index) => {
    html += `<div class="dimension-item" style="animation-delay: ${index * 0.1}s">
      <label class="form-label small fw-bold">${dim.label} <span class="text-danger">*</span></label>`;
    
    if (dim.possible_values && dim.possible_values.length > 0) {
      html += `<select class="dimension-select dimension-field" data-dim-key="${dim.key_name}" required>
        <option value="">-- Выберите ${dim.label.toLowerCase()} --</option>`;
      dim.possible_values.forEach(val => {
        html += `<option value="${val}">${val}</option>`;
      });
      html += `</select>`;
    } else {
      html += `<input type="text" class="dimension-select dimension-field" data-dim-key="${dim.key_name}" placeholder="Введите значение ${dim.label.toLowerCase()}" required>`;
    }
    
    if (dim.description) {
      html += `<small class="text-muted d-block mt-1">${dim.description}</small>`;
    }
    
    html += `</div>`;
  });
  
  container.innerHTML = html;
  
  // Добавляем слушатели для валидации
  setTimeout(() => {
    document.querySelectorAll('.dimension-field').forEach(el => {
      el.addEventListener('change', checkCanSave);
      el.addEventListener('input', checkCanSave);
    });
  }, 0);
}

// === LOADERS ===

async function loadMaterials() {
  const sel = document.getElementById('materialSelect');
  if (!sel) return;

  sel.innerHTML = '<option value="">-- Загрузка... --</option>';

  try {
    console.log('🔄 Loading materials...');
    const data = await api.getMaterials();
    const list = Array.isArray(data) ? data : (data?.items || []);

    sel.innerHTML = '<option value="">-- Выберите материал --</option>';

    if (list.length === 0) {
      sel.innerHTML += '<option value="" disabled>⚠️ Нет материалов в БД</option>';
      console.warn('⚠️ Materials list is empty');
      return;
    }

    list.forEach(m => {
      const id = m.id; 
      const name = m.name || 'Без названия';

      if (id) {
        const option = document.createElement('option');
        option.value = id;
        option.textContent = name;
        sel.appendChild(option);
        console.log(`  - Added: ${name} (${id})`);
      }
    });
    
    console.log(`✅ Loaded ${list.length} materials`);
  } catch (err) {
    console.error('❌ Failed to load materials:', err);
    sel.innerHTML = '<option value="">-- Ошибка загрузки --</option>';
    showToast('Не удалось загрузить материалы', true);
  }
}

async function loadGroups() {
  const sel = document.getElementById('groupSelect');
  if (!sel) return;

  try {
    const res = await api.getGroups('limit=10&offset=0');
    console.log('📦 Groups API response:', res);
    const list = res.items || (Array.isArray(res) ? res : []);
    console.log('📋 Groups list:', list);

    sel.innerHTML = '<option value="">-- Без группы --</option>';
    if (list.length === 0) {
      console.warn('⚠️ No groups found in database');
    }
    list.forEach(g => {
      const opt = document.createElement('option');
      opt.value = g.id;
      opt.textContent = g.name;
      sel.appendChild(opt);
    });
  } catch (err) {
    console.error('❌ Failed to load groups', err);
  }
}

async function loadEmployees() {
  const sel = document.getElementById('responsibleSelect');
  if (!sel) return;

  try {
    const employees = await api.getEmployees();
    console.log('👥 Employees loaded:', employees);

    sel.innerHTML = '<option value="">-- Без ответственного --</option>';
    if (!employees || employees.length === 0) {
      console.warn('⚠️ No employees found in database');
      return;
    }
    employees.forEach(emp => {
      const opt = document.createElement('option');
      opt.value = emp.id;
      opt.textContent = emp.name || emp.email || 'Сотрудник';
      sel.appendChild(opt);
    });
  } catch (err) {
    console.error('❌ Failed to load employees', err);
    sel.innerHTML = '<option value="">-- Ошибка загрузки --</option>';
  }
}

// === RENDER & LOGIC ===

function renderMethodDetails(full) {
  const method = full.method;
  const isCalc = method.formula_expr && full.inputs?.length > 0;
  
  let html = `<div class="method-box p-3 border rounded bg-light">
    <h5 class="mb-3">${method.name} ${method.unit ? `(${method.unit})` : ''}</h5>`;
  
  if (isCalc) {
    full.inputs.forEach(inp => {
      html += `
        <div class="mb-2">
          <label class="form-label small">${inp.label} ${inp.unit ? `(${inp.unit})` : ''} ${inp.is_required ? '<span class="text-danger">*</span>' : ''}</label>
          <input type="number" step="any" data-param="${inp.param_key}" class="form-control method-input" placeholder="0.00">
        </div>`;
    });
    if (method.formula_expr) {
      html += `<div class="alert alert-info small">Формула: <code>${method.formula_expr}</code></div>`;
    }
    html += `<div id="calcResult" class="fw-bold text-primary mt-2"></div>`;
    
    setTimeout(() => {
      document.querySelectorAll('.method-input').forEach(inp => {
        inp.addEventListener('input', calculatePreview);
      });
    }, 0);
  } else {
    html += `
      <div class="mb-2">
        <label class="form-label">Результат ${method.unit ? `(${method.unit})` : ''} <span class="text-danger">*</span></label>
        <input type="number" step="any" id="manualValue" class="form-control method-input">
      </div>`;
    setTimeout(() => {
      const el = document.getElementById('manualValue');
      if(el) el.addEventListener('input', checkCanSave);
    }, 0);
  }
  
  html += `
    <div class="form-group mt-3">
      <label class="small text-muted">Заметка к результату</label>
      <textarea id="resultNote" rows="2" class="form-control form-control-sm" placeholder="Комментарий к измерению..."></textarea>
    </div>`;
  
  html += `</div>`;
  document.getElementById('methodDetails').innerHTML = html;
  
  setTimeout(() => {
    const noteEl = document.getElementById('resultNote');
    if (noteEl) {
      noteEl.addEventListener('input', checkCanSave);
      noteEl.addEventListener('change', checkCanSave);
    }
  }, 0);
}

function calculatePreview() {
  if (!currentMethod?.formula_expr) return;
  
  const params = {};
  let allFilled = true;
  
  currentMethod.inputs.forEach(inp => {
    const el = document.querySelector(`[data-param="${inp.param_key}"]`);
    const val = el ? el.value : '';
    if (val) {
        params[inp.param_key] = parseFloat(val);
    } else if (inp.is_required) {
        allFilled = false;
    }
  });
  
  if (!allFilled) {
      const resEl = document.getElementById('calcResult');
      if (resEl) resEl.textContent = '';
      return;
  }

  try {
    let expr = currentMethod.formula_expr;
    for (const [k, v] of Object.entries(params)) {
      expr = expr.replace(new RegExp(`\\b${k}\\b`, 'g'), v);
    }
    
    if (/^[\d\s\+\-\*\/\.\(\)]+$/.test(expr)) {
      // eslint-disable-next-line no-new-func
      const result = Function('"use strict"; return (' + expr + ')')();
      const rounded = Math.round(result * 100) / 100;
      
      const resEl = document.getElementById('calcResult');
      const badgeEl = document.getElementById('calcBadge');
      
      if (resEl) resEl.textContent = `Расчет: ${rounded} ${currentMethod.unit || ''}`;
      if (badgeEl) {
          badgeEl.textContent = `Готов к сохранению`;
          badgeEl.className = 'badge badge-success';
      }
      checkCanSave();
    }
  } catch(e) {
    console.error('Calc error', e);
  }
}

function checkCanSave() {
  const requiredIds = ['sampleNumber', 'samplePlace', 'labName', 'operator', 'materialSelect', 'standardSelect', 'methodSelect'];
  let isValid = true;
  
  requiredIds.forEach(id => {
      const el = document.getElementById(id);
      if (!el || !el.value.trim()) isValid = false;
  });

  // 🔥 Проверяем, что все измерения заполнены
  document.querySelectorAll('.dimension-field').forEach(el => {
      if (!el.value.trim()) isValid = false;
  });

  if (isValid && currentMethod) {
      if (currentMethod.formula_expr) {
         const inputs = document.querySelectorAll('.method-input');
         inputs.forEach(inp => {
             if (!inp.value.trim()) isValid = false;
         });
      } else {
         const val = document.getElementById('manualValue')?.value;
         if (!val || isNaN(parseFloat(val))) isValid = false;
      }
  } else if (isValid && !currentMethod) {
      isValid = false;
  }
  
  const btn = document.getElementById('saveBtn');
  if (btn) btn.disabled = !isValid;
}

function resetMethodForm() {
  const mthSel = document.getElementById('methodSelect');
  const stdSel = document.getElementById('standardSelect');
  
  if (mthSel) {
      mthSel.value = '';
      mthSel.dispatchEvent(new Event('change'));
  }
  if (stdSel) {
      stdSel.value = '';
  }
  
  document.getElementById('methodDetails').innerHTML = '';
  currentMethod = null;
  standardDimensions = [];
  document.getElementById('dimensionsCard').style.display = 'none';
  checkCanSave();
}

async function saveProtocol() {
  if (!currentMethod) {
      showToast('Выберите метод испытания', true);
      return;
  }

  // 🔥 Сбор context_params из измерений
  const contextParams = {};
  let dimsValid = true;
  document.querySelectorAll('.dimension-field').forEach(el => {
      const key = el.dataset.dimKey;
      const val = el.value.trim();
      if (!val) {
          dimsValid = false;
      } else {
          contextParams[key] = val;
      }
  });
  
  if (!dimsValid && standardDimensions.length > 0) {
      showToast('Заполните все параметры контекста', true);
      return;
  }

  const sampleNoteEl = document.getElementById('sampleNote');
  const protocolNoteEl = document.getElementById('protocolNote');
  const resultNoteEl = document.getElementById('resultNote');
  const collectionDateEl = document.getElementById('collectionDate');
  
  const sampleNote = sampleNoteEl ? (sampleNoteEl.value.trim() || null) : null;
  const protocolNote = protocolNoteEl ? (protocolNoteEl.value.trim() || null) : null;
  const resultNote = resultNoteEl ? (resultNoteEl.value.trim() || null) : null;
  
  let collectionDate = null;
  if (collectionDateEl && collectionDateEl.value) {
    collectionDate = new Date(collectionDateEl.value + 'T00:00:00Z').toISOString();
  }

  // 🔥 Сбор расширенных параметров образца
  const lengthMM = document.getElementById('lengthMM')?.value ? parseFloat(document.getElementById('lengthMM').value) : null;
  const widthMM = document.getElementById('widthMM')?.value ? parseFloat(document.getElementById('widthMM').value) : null;
  const heightMM = document.getElementById('heightMM')?.value ? parseFloat(document.getElementById('heightMM').value) : null;
  const weightGrams = document.getElementById('weightGrams')?.value ? parseFloat(document.getElementById('weightGrams').value) : null;
  const shape = document.getElementById('shape')?.value || '';
  const color = document.getElementById('color')?.value.trim() || '';
  const batchNumber = document.getElementById('batchNumber')?.value.trim() || '';
  const manufacturer = document.getElementById('manufacturer')?.value.trim() || '';
  const photoURL = document.getElementById('photoURL')?.value.trim() || '';

  const results = [];
  let rawInputs = {};

  if (currentMethod.formula_expr) {
     currentMethod.inputs.forEach(inp => {
        const el = document.querySelector(`[data-param="${inp.param_key}"]`);
        if (el) rawInputs[inp.param_key] = el.value;
     });
  } else {
     const val = document.getElementById('manualValue')?.value;
     if (!val || isNaN(parseFloat(val))) {
        showToast('Введите результат испытания', true);
        return;
     }
     rawInputs = { value: val };
  }

  results.push({ 
      method_id: currentMethod.id, 
      raw_inputs: rawInputs,
      note: resultNote
  });

  const payload = {
    group_id: document.getElementById('groupSelect').value || null,
    sample: {
      sample_number: document.getElementById('sampleNumber').value,
      material_id: document.getElementById('materialSelect').value,
      collection_place: document.getElementById('samplePlace').value,
      collection_date: collectionDate,
      context_params: contextParams, // 🔥 Передаем собранные параметры контекста
      note: sampleNote,
      // Расширенные поля образца
      length_mm: lengthMM,
      width_mm: widthMM,
      height_mm: heightMM,
      weight_grams: weightGrams,
      shape: shape,
      color: color,
      batch_number: batchNumber,
      manufacturer: manufacturer,
      photo_url: photoURL
    },
    lab_name: document.getElementById('labName').value,
    operator_name: document.getElementById('operator').value,
    responsible_person_id: document.getElementById('responsibleSelect')?.value || null,
    note: protocolNote,
    results: results
  };

  try {
    const btn = document.getElementById('saveBtn');
    setLoading(btn, true);
    await api.createProtocol(payload);
    showToast('✅ Протокол успешно сохранен!');
    setTimeout(() => window.location.href = '/protocols.html', 1000);
  } catch (e) {
    console.error('Save error:', e);
    showToast('❌ Ошибка сохранения: ' + (e.message || 'Неизвестная ошибка'), true);
  } finally {
    const btn = document.getElementById('saveBtn');
    setLoading(btn, false);
  }
}

window.resetMethodForm = resetMethodForm;
window.saveProtocol = saveProtocol;