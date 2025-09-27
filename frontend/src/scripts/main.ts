import { logout, login } from '../services/api';
import { showNotification } from '../utils/notifications';

const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

/**
 * Handles all delegated click events for the entire application.
 * This single event listener is robust against Astro's View Transitions.
 */
document.addEventListener('click', async (e) => {
    const target = e.target as HTMLElement;

    // --- Logout Link ---
    if (target.matches('#logout-link')) {
        e.preventDefault();
        try {
            await logout();
            window.location.href = '/';
        } catch (error) {
            console.error('Logout failed:', error);
            showNotification({ message: '登出失败', type: 'error' });
        }
    }

    // --- Theme Toggle ---
    if (target.matches('#theme-toggle, #theme-toggle *')) {
        e.preventDefault();
        const htmlEl = document.documentElement;
        const currentTheme = htmlEl.classList.contains("dark") ? "dark" : "light";
        const newTheme = currentTheme === "dark" ? "light" : "dark";
        htmlEl.classList.remove("light", "dark");
        htmlEl.classList.add(newTheme);
        localStorage.setItem("theme", newTheme);
    }

    // --- Modal Triggers ---
    const modalTrigger = target.closest('[data-modal-target]');
    if (modalTrigger) {
        const modalId = modalTrigger.getAttribute('data-modal-target');
        const modal = document.getElementById(modalId!);
        if (modal) {
            modal.style.display = 'flex';
        }
    }

    // --- Modal Close Buttons ---
    const modalCloseBtn = target.closest('.modal-close-btn');
    if (modalCloseBtn) {
        const modal = modalCloseBtn.closest('.modal-container') as HTMLElement;
        if (modal) {
            modal.style.display = 'none';
        }
    }
    
    // --- Click outside Modal to close ---
    if (target.matches('.modal-container')) {
        target.style.display = 'none';
    }

    // --- Settings Page: Save Site Info ---
    if (target.matches('#save-settings-btn')) {
        const form = document.getElementById('settings-form') as HTMLFormElement;
        const formData = new FormData(form);
        const data: { [key: string]: any } = {};
        formData.forEach((value, key) => { data[key] = value; });
        if (!data.password) delete data.password;

        try {
            const response = await fetch(`${apiBaseUrl}/api/settings/`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify(data),
            });
            const result = await response.json();
            showNotification({ message: result.message || 'Error', type: response.ok ? 'success' : 'error' });
        } catch (error: any) {
            showNotification({ message: error.message, type: 'error' });
        }
    }
    
    // --- Settings Page: Save AI Settings ---
    if (target.matches('#save-ai-btn')) {
        const form = document.getElementById('ai-settings-form') as HTMLFormElement;
        const modal = document.getElementById('ai-modal') as HTMLElement;
        const formData = new FormData(form);
        const data: { [key: string]: any } = {};
        formData.forEach((value, key) => { data[key] = value; });
        if (!data.openai_token) delete data.openai_token;

        try {
            const response = await fetch(`${apiBaseUrl}/api/settings/`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify(data),
            });
            const result = await response.json();
            showNotification({ message: result.message || 'Error', type: response.ok ? 'success' : 'error' });
            if (response.ok) modal.style.display = 'none';
        } catch (error: any) {
            showNotification({ message: error.message, type: 'error' });
        }
    }
});

/**
 * Handles all delegated form submissions.
 */
document.addEventListener('submit', async (e) => {
    // --- Login Form ---
    if (e.target && (e.target as HTMLElement).id === 'login-form') {
        e.preventDefault();
        const form = e.target as HTMLFormElement;
        const passwordInput = form.querySelector('#password') as HTMLInputElement;
        const password = passwordInput.value;

        try {
            const response = await login(password);
            if (response.status === 'success') {
                showNotification({ message: '登录成功!', type: 'success' });
                window.location.href = '/admin';
            } else {
                showNotification({ message: response.message || '登录失败', type: 'error' });
            }
        } catch (error) {
            if (error instanceof Error) {
                showNotification({ message: error.message, type: 'error' });
            } else {
                showNotification({ message: 'An unknown error occurred.', type: 'error' });
            }
        }
    }
});

/**
 * Handles page-specific setup that can't be purely delegated.
 */
function initializePage() {
    // --- Back to Top Button ---
    const backToTopButton = document.getElementById('back-to-top');
    if (backToTopButton) {
        const scrollHandler = () => {
            if (window.pageYOffset > 200) {
                backToTopButton.classList.add('show');
            } else {
                backToTopButton.classList.remove('show');
            }
        };
        // Attach listener only once
        if (!(window as any).scrollListenerAttached) {
            window.addEventListener('scroll', scrollHandler);
            (window as any).scrollListenerAttached = true;
        }
    }
}

document.addEventListener('DOMContentLoaded', initializePage);
document.addEventListener('astro:after-swap', initializePage);