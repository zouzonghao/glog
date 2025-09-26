import { logout } from '@/services/api';

// --- Logout Logic ---
async function handleLogout(e: Event) {
    e.preventDefault();
    try {
        await logout();
        window.location.href = '/login';
    } catch (error) {
        console.error('Logout failed:', error);
        // Optionally show a notification to the user
    }
}

// --- Notification Logic (from old main.js) ---
function showNotification(message: string, type: 'info' | 'success' | 'error' = 'info') {
    const container = document.getElementById('notification-container');
    if (!container) {
        console.error('Notification container not found.');
        return;
    }

    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;

    container.appendChild(notification);

    setTimeout(() => {
        notification.classList.add('show');
    }, 10);

    setTimeout(() => {
        notification.classList.remove('show');
        notification.addEventListener('transitionend', () => {
            notification.remove();
        });
    }, 5000);
}

// --- DOMContentLoaded Logic ---
document.addEventListener('DOMContentLoaded', function() {
    // --- Logout Link ---
    const logoutLink = document.getElementById('logout-link');
    if (logoutLink) {
        logoutLink.addEventListener('click', handleLogout);
    }

    // --- Theme Toggle ---
    const themeToggle = document.getElementById("theme-toggle");
    const htmlEl = document.documentElement;

    const setTheme = (theme: string) => {
        htmlEl.classList.remove("light", "dark");
        htmlEl.classList.add(theme);
        localStorage.setItem("theme", theme);
    };

    if (themeToggle) {
        themeToggle.addEventListener("click", () => {
            const currentTheme = htmlEl.classList.contains("dark") ? "dark" : "light";
            const newTheme = currentTheme === "dark" ? "light" : "dark";
            setTheme(newTheme);
        });
    }

    // --- Back to Top Button ---
    const backToTopButton = document.getElementById('back-to-top');
    if (backToTopButton) {
        window.addEventListener('scroll', function() {
            if (window.pageYOffset > 200) {
                backToTopButton.classList.add('show');
            } else {
                backToTopButton.classList.remove('show');
            }
        });
    }
});