// frontend/js/navigation.js

export function initNavigation() {
  const currentPath = window.location.pathname;
  const navLinks = document.querySelectorAll('.nav-link');
  
  navLinks.forEach(link => {
    const href = link.getAttribute('href');
    if (currentPath.endsWith(href) || (href === '/' && currentPath === '/')) {
      link.classList.add('active');
    }
  });
}

export function navigateTo(page) {
  window.location.href = page;
}