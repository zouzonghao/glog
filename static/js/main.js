// 全局可用的通知函数
function showNotification(message, type = 'info') {
    const container = document.getElementById('notification-container');
    if (!container) {
        console.error('Notification container not found.');
        return;
    }

    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;

    container.appendChild(notification);

    // Animate in
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);

    // 5秒后自动移除
    setTimeout(() => {
        notification.classList.remove('show');
        notification.addEventListener('transitionend', () => {
            notification.remove();
        });
    }, 5000);
}

// DOM 加载完成后执行的脚本
document.addEventListener('DOMContentLoaded', function() {
    // Auto-focus search bar on home page
    if (document.getElementById('home-page')) {
        const searchInput = document.querySelector('.search-input');
        if (searchInput) {
            searchInput.focus();
        }
    }
    

    // 主题切换逻辑
    const themeToggle = document.getElementById("theme-toggle");
    const htmlEl = document.documentElement;

    const setTheme = (theme) => {
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

    // 返回顶部按钮逻辑
    const backToTopButton = document.getElementById('back-to-top');
    if (backToTopButton) {
        window.addEventListener('scroll', function() {
            if (window.pageYOffset > 200) { // 滚动200px后显示
                backToTopButton.classList.add('show');
            } else {
                backToTopButton.classList.remove('show');
            }
        });
    }
});
// 全局可用的模态框设置函数
function setupGlobalModal(modalId, openTriggerId, closeTriggers = []) {
    const modal = document.getElementById(modalId);
    const openTrigger = document.getElementById(openTriggerId);

    if (!modal || !openTrigger) {
        console.warn(`Modal or open trigger not found for modalId: ${modalId}`);
        return;
    }

    const showModal = () => {
        modal.classList.add('show');
    };

    const hideModal = () => {
        modal.classList.remove('show');
    };

    openTrigger.addEventListener('click', showModal);

    // Add close triggers
    const allCloseTriggers = [...closeTriggers, ...modal.querySelectorAll('.modal-close-btn, .modal-cancel-btn')];
    allCloseTriggers.forEach(trigger => {
        const el = (typeof trigger === 'string') ? document.getElementById(trigger) : trigger;
        if (el) {
            el.addEventListener('click', hideModal);
        }
    });

    // Close when clicking on the background
    modal.addEventListener('click', (event) => {
        if (event.target === modal) {
            hideModal();
        }
    });
}
function showGlobalPasswordPrompt(title) {
    return new Promise((resolve) => {
        const modal = document.getElementById('password-prompt-modal');
        if (!modal) {
            console.error('Password prompt modal not found!');
            resolve(null);
            return;
        }

        const form = modal.querySelector('form');
        const titleEl = document.getElementById('password-prompt-title');
        const inputEl = document.getElementById('password-prompt-input');
        const confirmBtn = document.getElementById('password-prompt-confirm-btn');
        const cancelBtn = document.getElementById('password-prompt-cancel-btn');

        titleEl.textContent = title;
        inputEl.value = '';

        const cleanup = () => {
            confirmBtn.onclick = null;
            cancelBtn.onclick = null;
            form.onsubmit = null;
            modal.classList.remove('show');
        };

        const closeModal = (value) => {
            cleanup();
            resolve(value);
        };

        const submitHandler = (event) => {
            event.preventDefault();
            closeModal(inputEl.value);
        };

        confirmBtn.onclick = () => submitHandler(new Event('submit'));
        cancelBtn.onclick = () => closeModal(null);
        form.onsubmit = submitHandler;

        modal.classList.add('show');
        inputEl.focus();
    });
}