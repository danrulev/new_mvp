// frontend/js/theme.js
// Переключение темы (светлая/тёмная)

function toggleTheme() {
  const html = document.documentElement;
  const currentTheme = html.getAttribute('data-theme');
  const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
  
  html.setAttribute('data-theme', newTheme);
  localStorage.setItem('theme', newTheme);
  
  // Обновляем иконку кнопки
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.textContent = newTheme === 'dark' ? '☀️' : '🌙';
  }
}

// Инициализация темы при загрузке
document.addEventListener('DOMContentLoaded', () => {
  const savedTheme = localStorage.getItem('theme') || 'light';
  document.documentElement.setAttribute('data-theme', savedTheme);
  
  // Обновляем иконки на всех страницах
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.textContent = savedTheme === 'dark' ? '☀️' : '🌙';
  }
});
