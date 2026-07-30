// frontend/js/utils.js

export function formatDate(date) {
  if (!date) return '—';
  try {
    return new Date(date).toLocaleDateString('ru-RU', {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit'
    });
  } catch { return String(date); }
}

export function formatDateOnly(isoString) {
  if (!isoString) return '—';
  try {
    const datePart = isoString.split('T')[0];
    const [year, month, day] = datePart.split('-');
    return `${day}.${month}.${year}`;
  } catch {
    return '—';
  }
}

export function formatNumber(value, unit = '') {
  if (value == null || isNaN(value)) return '—';
  return `${Number(value).toFixed(2)}${unit ? ` ${unit}` : ''}`;
}

export function getNormString(limit) {
  if (!limit?.limit_type) return '—';
  const { limit_type, min_value, max_value } = limit;
  if (limit_type === 'range' && min_value != null && max_value != null) {
    return `${min_value.toFixed(2)} – ${max_value.toFixed(2)}`;
  }
  if (limit_type === 'min' && min_value != null) return `≥ ${min_value.toFixed(2)}`;
  if (limit_type === 'max' && max_value != null) return `≤ ${max_value.toFixed(2)}`;
  return '—';
}

export function showToast(message, error = false) {
  const toast = document.createElement('div');
  toast.className = `toast${error ? ' error' : ''}`;
  toast.innerHTML = `
    <span>${error ? '❌' : '✅'}</span>
    <span style="flex:1">${message}</span>
    <button class="close">&times;</button>
  `;
  document.body.appendChild(toast);
  
  toast.querySelector('.close').onclick = () => toast.remove();
  setTimeout(() => toast.remove(), error ? 5000 : 3000);
}

export function showConfirm(message) {
  return new Promise(resolve => {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
      <div class="modal-content">
        <div class="modal-header">
          <h3>Подтверждение</h3>
          <button class="modal-close">&times;</button>
        </div>
        <div class="modal-body">${message}</div>
        <div class="modal-footer">
          <button class="btn btn-secondary" id="cancel">Отмена</button>
          <button class="btn btn-danger" id="confirm">Подтвердить</button>
        </div>
      </div>
    `;
    document.body.appendChild(modal);
    
    const close = () => { modal.remove(); resolve(false); };
    modal.querySelector('.modal-close').onclick = close;
    modal.querySelector('#cancel').onclick = close;
    modal.querySelector('#confirm').onclick = () => { modal.remove(); resolve(true); };
    modal.onclick = (e) => { if (e.target === modal) close(); };
  });
}

export function setLoading(el, loading) {
  if (loading) {
    el.classList.add('loading');
    el.dataset.disabled = el.disabled;
    el.disabled = true;
  } else {
    el.classList.remove('loading');
    el.disabled = el.dataset.disabled === 'true';
  }
}