import { logout } from '@/services/api';

// --- Logout Logic ---
async function handleLogout(e: Event) {
    e.preventDefault();
    try {
        await logout();
        window.location.href = '/';
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
function initializePage() {
    // --- Logout Link ---
    const logoutLink = document.getElementById('logout-link');
    if (logoutLink) {
        // To prevent multiple listeners, we can remove it before adding it,
        // though for a simple logout, it might not be strictly necessary.
        logoutLink.removeEventListener('click', handleLogout);
        logoutLink.addEventListener('click', handleLogout);
    }

    // --- Theme Toggle ---
    const themeToggle = document.getElementById("theme-toggle");
    const htmlEl = document.documentElement;

    // Ensure the correct theme is applied visually on load/swap
    const savedTheme = localStorage.getItem("theme") || (window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
    htmlEl.classList.remove("light", "dark");
    htmlEl.classList.add(savedTheme);

    const setTheme = (theme: string) => {
        htmlEl.classList.remove("light", "dark");
        htmlEl.classList.add(theme);
        localStorage.setItem("theme", theme);
    };

    if (themeToggle) {
        // A simple way to avoid multiple listeners is to replace the element
        // This is a bit of a hack, but effective for this case.
        const newToggle = themeToggle.cloneNode(true);
        themeToggle.parentNode?.replaceChild(newToggle, themeToggle);
        
        newToggle.addEventListener("click", () => {
            const currentTheme = htmlEl.classList.contains("dark") ? "dark" : "light";
            const newTheme = currentTheme === "dark" ? "light" : "dark";
            setTheme(newTheme);
        });
    }

    // --- Back to Top Button ---
    const backToTopButton = document.getElementById('back-to-top');
    if (backToTopButton) {
        // The scroll listener is on window, so we don't need to re-add it
        // if it's already there. A simple flag can handle this.
        if (!(window as any).scrollListenerAttached) {
            window.addEventListener('scroll', function() {
                if (window.pageYOffset > 200) {
                    backToTopButton.classList.add('show');
                } else {
                    backToTopButton.classList.remove('show');
                }
            });
            (window as any).scrollListenerAttached = true;
        }
    }
}

// --- Event Listeners ---
// Run on initial page load
document.addEventListener('DOMContentLoaded', initializePage);
// Run after every Astro view transition
document.addEventListener('astro:after-swap', initializePage);