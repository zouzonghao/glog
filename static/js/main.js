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

    // 代码块复制功能
    const codeBlocks = document.querySelectorAll('.post-content pre');
    codeBlocks.forEach(pre => {
        const code = pre.querySelector('code');
        if (!code) return;

        const wrapper = document.createElement('div');
        wrapper.className = 'code-wrapper';
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);

        const copyBtn = document.createElement('button');
        copyBtn.className = 'copy-btn';
        copyBtn.textContent = '复制';
        copyBtn.setAttribute('type', 'button');
        wrapper.appendChild(copyBtn);

        copyBtn.addEventListener('click', async () => {
            const text = code.textContent;
            try {
                await navigator.clipboard.writeText(text);
                copyBtn.textContent = '已复制';
                copyBtn.classList.add('copied');
                setTimeout(() => {
                    copyBtn.textContent = '复制';
                    copyBtn.classList.remove('copied');
                }, 2000);
            } catch (err) {
                console.error('复制失败:', err);
                copyBtn.textContent = '失败';
                setTimeout(() => {
                    copyBtn.textContent = '复制';
                }, 2000);
            }
        });

        const classList = Array.from(pre.classList);
        const langClass = classList.find(cls => cls.startsWith('language-'));
        if (langClass) {
            const lang = langClass.replace('language-', '');
            pre.setAttribute('data-lang', lang);
        } else {
            pre.setAttribute('data-lang', 'Code');
        }
    });

    // 返回顶部按钮逻辑 (使用 IntersectionObserver 优化性能)
    const backToTopButton = document.getElementById('back-to-top');
    const topSentinel = document.getElementById('top-sentinel') || document.body; // 优先使用哨兵，降级使用 body

    if (backToTopButton) {
        const observer = new IntersectionObserver((entries) => {
            // 当顶部元素(header/body)离开视口时，entries[0].isIntersecting 变为 false
            // 此时应该显示返回顶部按钮
            if (!entries[0].isIntersecting) {
                backToTopButton.classList.add('show');
            } else {
                backToTopButton.classList.remove('show');
            }
        }, {
            root: null,
            threshold: 0,
            rootMargin: "200px 0px 0px 0px" // 向下偏移200px才触发，模拟之前的 scroll > 200
        });

        observer.observe(topSentinel);
        
        // 点击平滑滚动回顶部
        backToTopButton.addEventListener('click', (e) => {
            e.preventDefault();
            window.scrollTo({ top: 0, behavior: 'smooth' });
        });
    }

    // 页码控制栏显示逻辑 - 鼠标靠近时显示
    const pagination = document.querySelector('.pagination-new');
    if (pagination) {
        let mouseHandler = null;
        const proximityThreshold = 300; // 鼠标靠近150px时触发

        const handleMouseMove = (e) => {
            const rect = pagination.getBoundingClientRect();
            const mouseX = e.clientX;
            const mouseY = e.clientY;
            
            // 计算鼠标到页码控制栏的距离
            const distanceX = Math.max(0, Math.max(rect.left - mouseX, mouseX - rect.right));
            const distanceY = Math.max(0, Math.max(rect.top - mouseY, mouseY - rect.bottom));
            const distance = Math.sqrt(distanceX * distanceX + distanceY * distanceY);
            
            // 当鼠标靠近时显示
            if (distance < proximityThreshold) {
                pagination.classList.add('visible');
                // 显示后移除监听器
                window.removeEventListener('mousemove', mouseHandler);
            }
        };

        // 保存监听器引用以便移除
        mouseHandler = handleMouseMove;
        
        // 监听鼠标移动事件
        window.addEventListener('mousemove', mouseHandler);
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