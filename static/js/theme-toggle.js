const themeToggle = document.getElementById('theme-toggle');
const htmlEl = document.documentElement;

// Function to set theme
const setTheme = (theme) => {
    htmlEl.classList.remove('light', 'dark');
    htmlEl.classList.add(theme);
    localStorage.setItem('theme', theme);
};

// Check for saved theme in localStorage or system preference
const savedTheme = localStorage.getItem('theme');
const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

if (savedTheme) {
    setTheme(savedTheme);
} else if (prefersDark) {
    setTheme('dark');
} else {
    setTheme('light');
}

// Event listener for the toggle button
themeToggle.addEventListener('click', () => {
    const currentTheme = htmlEl.classList.contains('dark') ? 'dark' : 'light';
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    setTheme(newTheme);
});