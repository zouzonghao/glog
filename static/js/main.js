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