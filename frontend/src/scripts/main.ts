import { logout, login } from '../services/api';
import { showNotification } from '../utils/notifications';

const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

// =================================================================================
// Global Event Listeners
// =================================================================================

/**
 * Handles globally delegated click events.
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
                // Use Astro's client-side router for smoother navigation
                (window as any).Astro.navigate('/admin');
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