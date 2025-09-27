const applyTheme = (theme: string) => {
  // Explicitly remove both classes and add the correct one.
  document.documentElement.classList.remove('light', 'dark');
  document.documentElement.classList.add(theme);
};

const initTheme = () => {
  // 1. Restore theme from localStorage on every page load/transition
  const savedTheme = localStorage.getItem('theme');
  // If a theme is saved in localStorage, apply it.
  // The inline script has already handled the initial load based on system preference,
  // so we only need to ensure the localStorage state is respected after swaps.
  if (savedTheme) {
    applyTheme(savedTheme);
  }

  // 2. Setup the click listener for the toggle button
  const themeToggleButton = document.getElementById('theme-toggle');
  if (!themeToggleButton) {
    return;
  }

  const handleClick = (e: Event) => {
    e.preventDefault();
    const currentTheme = document.documentElement.classList.contains('dark') ? 'dark' : 'light';
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    
    localStorage.setItem('theme', newTheme);
    applyTheme(newTheme);
  };

  // Use a flag to prevent adding the listener multiple times on the same element
  if (!(themeToggleButton as any).themeListenerAttached) {
    themeToggleButton.addEventListener('click', handleClick);
    (themeToggleButton as any).themeListenerAttached = true;
  }
};

// Run on initial page load
document.addEventListener('DOMContentLoaded', initTheme);

// Run after every Astro View Transition
document.addEventListener('astro:after-swap', initTheme);