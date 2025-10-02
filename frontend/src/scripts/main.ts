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
    // Login form logic is now handled in login.astro directly.
});

/**
 * Handles page-specific setup that can't be purely delegated.
 */
/**
 * Handles page-specific setup that can't be purely delegated.
 */
// Define the scroll handler at a higher scope so it can be referenced for removal.
let scrollHandler: () => void;

function initializePage() {
    // --- Back to Top Button ---
    const backToTopButton = document.getElementById('back-to-top');

    // Clean up the old listener before attaching a new one.
    if (scrollHandler) {
        window.removeEventListener('scroll', scrollHandler);
    }

    if (backToTopButton) {
        scrollHandler = () => {
            if (window.pageYOffset > 200) {
                backToTopButton.classList.add('show');
            } else {
                backToTopButton.classList.remove('show');
            }
        };
        window.addEventListener('scroll', scrollHandler);
        // Also, trigger it once on load to set the initial state.
        scrollHandler();
    }
}

document.addEventListener('DOMContentLoaded', initializePage);
document.addEventListener('astro:after-swap', initializePage);