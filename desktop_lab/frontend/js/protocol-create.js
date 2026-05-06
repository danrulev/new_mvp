// frontend/js/protocol-create.js
import { api } from './api.js';
import { showToast, setLoading } from './utils.js';
import { initNavigation } from './navigation.js';

// === STATE ===
let standards = [], methods = [];
let currentMethod = null;

// === INIT ===
document.addEventListener('DOMContentLoaded', async () => {
  console.log('🚀 DOM Loaded, initializing...');
  
  // 1. Инициализация навигации
  initNavigation();
  
  // 2. Настройка слушателей событий
  setupFormListeners();
  setupCascadingSelects();
  
  // 3. Загрузка данных
  await loadMaterials();
  await loadGroups();
});

function setupFormListeners() {
  const inputs = ['sampleNumber', 'samplePlace', 'labName', 'operator'];
  inputs.forEach(id => {
    const el = document.getElementById(id);
    if (el) {
      el.addEventListener('input', checkCanSave);
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
    
    // Сброс зависимых полей
    stdSel.value = '';
    mthSel.value = '';
    mthSel.innerHTML = '<option value="">-- Выберите стандарт --</option>';
    mthSel.disabled = true;
    document.getElementById('methodDetails').innerHTML = '';
    currentMethod = null;
    checkCanSave();

    if (!matId) {
      stdSel.innerHTML = '<option value="">-- Выберите материал --</option>';
      stdSel.disabled = true;
      return;
    }

    stdSel.disabled = false;
    stdSel.innerHTML = '<option value="">-- Загрузка... --</option>';

    try {
      standards = await api.getStandardsByMaterial(matId);
      stdSel.innerHTML = '<option value="">-- Выберите стандарт --</option>';
      
      if (!standards || standards.length === 0) {
        stdSel.innerHTML += '<option value="" disabled>Нет стандартов</option>';
      } else {
        standards.forEach(s => {
          const opt = document.createElement('option');
          opt.value = s.id;
          opt.textContent = s.name;
          stdSel.appendChild(opt);
        });
      }
    } catch (err) {
      console.error('Error loading standards:', err);
      stdSel.innerHTML = '<option value="">Ошибка загрузки</option>';
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
      return;
    }

    mthSel.disabled = false;
    mthSel.innerHTML = '<option value="">-- Загрузка... --</option>';

    try {
      methods = await api.getMethodsByStandard(stdId);
      mthSel.innerHTML = '<option value="">-- Выберите метод --</option>';
      
      methods.forEach(m => {
        const opt = document.createElement('option');
        opt.value = m.id;
        opt.textContent = `${m.name}${m.unit ? ` (${m.unit})` : ''}`;
        mthSel.appendChild(opt);
      });
    } catch (err) {
      console.error('Error loading methods:', err);
      mthSel.innerHTML = '<option value="">Ошибка загрузки</option>';
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
      detailsContainer.innerHTML = '<div class="text-danger">Ошибка загрузки метода</div>';
    }
  });
}

// === LOADERS ===

async function loadMaterials() {
  const sel = document.getElementById('materialSelect');
  if (!sel) return;

  sel.innerHTML = '<option value="">-- Загрузка... --</option>';

  try {
    console.log('🔄 Loading materials...');
    const data = await api.getMaterials();
    console.log('✅ Raw API Response:', data);

    // Приводим к массиву (на случай если бэкенд вернет объект с полем items)
    const list = Array.isArray(data) ? data : (data.items || []);

    sel.innerHTML = '<option value="">-- Выберите материал --</option>';

    if (list.length === 0) {
      sel.innerHTML += '<option value="" disabled>Нет материалов в БД</option>';
      console.warn('⚠️ Materials list is empty');
      return;
    }

    list.forEach(m => {
      // Используем поля в нижнем регистре, так как в Go стоят теги json:"id"
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
    const res = await api.getGroups(100, 0);
    // Проверка структуры ответа для групп
    const list = res.items || (Array.isArray(res) ? res : []);
    
    sel.innerHTML = '<option value="">-- Без группы --</option>';
    list.forEach(g => {
      const opt = document.createElement('option');
      opt.value = g.id;
      opt.textContent = g.name;
      sel.appendChild(opt);
    });
  } catch (err) {
    console.error('Failed to load groups', err);
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
        <label class="form-label">Результат ${method.unit ? `(${method.unit})` : ''}</label>
        <input type="number" step="any" id="manualValue" class="form-control method-input">
      </div>`;
    setTimeout(() => {
      const el = document.getElementById('manualValue');
      if(el) el.addEventListener('input', checkCanSave);
    }, 0);
  }
  
  html += `</div>`;
  document.getElementById('methodDetails').innerHTML = html;
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
  if (mthSel) {
      mthSel.value = '';
      mthSel.dispatchEvent(new Event('change'));
  }
}

async function saveProtocol() {
  const btn = document.getElementById('saveBtn');
  if (!currentMethod) {
      showToast('Выберите метод испытания', true);
      return;
  }

  const results = [];
  let rawInputs = {};

  if (currentMethod.formula_expr) {
     currentMethod.inputs.forEach(inp => {
        const el = document.querySelector(`[data-param="${inp.param_key}"]`);
        if (el) rawInputs[inp.param_key] = el.value;
     });
  } else {
     const val = document.getElementById('manualValue')?.value;
     rawInputs = { value: val };
  }

  results.push({ 
      method_id: currentMethod.id, 
      raw_inputs: rawInputs 
  });

  const payload = {
    group_id: document.getElementById('groupSelect').value || null,
    sample: {
      sample_number: document.getElementById('sampleNumber').value,
      material_id: document.getElementById('materialSelect').value,
      collection_place: document.getElementById('samplePlace').value,
      context_params: {},
      note: null
    },
    lab_name: document.getElementById('labName').value,
    operator_name: document.getElementById('operator').value,
    results: results
  };

  try {
    setLoading(btn, true);
    await api.createProtocol(payload);
    showToast('Протокол успешно сохранен!');
    setTimeout(() => window.location.href = '/protocols.html', 1000);
  } catch (e) {
    console.error(e);
  } finally {
    setLoading(btn, false);
  }
}

// Экспорт для HTML onclick
window.resetMethodForm = resetMethodForm;
window.saveProtocol = saveProtocol;