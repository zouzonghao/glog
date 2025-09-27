import { showNotification } from '../utils/notifications';

const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

// Helper functions specific to the settings page

function updateImageApiModelSelector(providers: { provider: string, models: { name: string }[] }[]) {
    const container = document.getElementById('imageapi-model-container');
    if (!container) return;

    const currentInput = container.querySelector('input[name="imageapi_model"]') as HTMLInputElement;
    const currentModel = currentInput?.value;

    const select = document.createElement('select');
    select.id = 'imageapi_model';
    select.name = 'imageapi_model';

    providers.forEach(provider => {
        const optgroup = document.createElement('optgroup');
        optgroup.label = provider.provider;
        provider.models.forEach(model => {
            const option = document.createElement('option');
            option.value = model.name;
            option.textContent = model.name;
            if (model.name === currentModel) {
                option.selected = true;
            }
            optgroup.appendChild(option);
        });
        select.appendChild(optgroup);
    });

    const label = container.querySelector('label');
    container.innerHTML = '';
    if (label) container.appendChild(label);
    container.appendChild(select);
}

async function saveSettings(form: HTMLFormElement) {
    const modal = form.closest('.modal-container');
    const formData = new FormData(form);
    const data: { [key: string]: any } = {};
    formData.forEach((value, key) => { data[key] = value; });

    ['password', 'openai_token', 'imageapi_token', 'github_token', 'webdav_password'].forEach(key => {
        if (data[key] === '') delete data[key];
    });

    try {
        const response = await fetch(`${apiBaseUrl}/api/settings/`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(data),
        });
        const result = await response.json();
        showNotification({ message: result.message || 'Error', type: response.ok ? 'success' : 'error' });
        if (response.ok && modal) modal.classList.remove('show');
    } catch (error: any) {
        showNotification({ message: error.message, type: 'error' });
    }
}

async function testConnection(form: HTMLFormElement, type: 'ai' | 'imageapi' | 'github' | 'webdav') {
    const button = form.querySelector(`#test-${type}-btn`) as HTMLButtonElement;
    if (button) button.disabled = true;
    showNotification({ message: '测试中...', type: 'info' });

    try {
        const response = await fetch(`${apiBaseUrl}/api/settings/test-${type}`, {
            method: 'POST',
            credentials: 'include',
            body: new URLSearchParams(new FormData(form) as any),
        });
        const result = await response.json();
        let message = result.message || 'Error';
        if (type === 'imageapi' && result.models && result.models.length > 0) {
            updateImageApiModelSelector(result.models);
            message += ` (已更新模型列表)`;
        }
        showNotification({ message, type: result.status === 'success' ? 'success' : 'error' });
    } catch (error: any) {
        showNotification({ message: error.message, type: 'error' });
    } finally {
        if (button) button.disabled = false;
    }
}

async function backupNow(type: 'github' | 'webdav') {
    const button = document.getElementById(`backup-${type}-now-btn`) as HTMLButtonElement;
    if(button) button.disabled = true;
    showNotification({ message: '正在备份...', type: 'info' });

    try {
        const response = await fetch(`${apiBaseUrl}/api/backup/${type}-now`, {
            method: 'POST',
            credentials: 'include',
        });
        const result = await response.json();
        showNotification({ message: result.message, type: result.status });
    } catch (error: any) {
        showNotification({ message: error.message, type: 'error' });
    } finally {
        if(button) button.disabled = false;
    }
}

async function fetchAndDisplayAILogs() {
    const logsContent = document.getElementById('ai-logs-content');
    if (!logsContent) return;
    logsContent.textContent = '正在加载日志...';
    try {
        const response = await fetch(`${apiBaseUrl}/api/ai-logs/`, { credentials: 'include' });
        const result = await response.json();
        logsContent.textContent = result.status === 'success' ? (result.logs || '暂无 AI 日志。') : `加载日志失败: ${result.message}`;
    } catch (error: any) {
        logsContent.textContent = `加载日志时发生网络错误: ${error.message}`;
    }
}

async function clearAILogs() {
    showNotification({ message: '正在清除日志...', type: 'info' });
    try {
        const response = await fetch(`${apiBaseUrl}/api/ai-logs/clear`, {
            method: 'POST',
            credentials: 'include',
        });
        const result = await response.json();
        showNotification({ message: result.message, type: result.status });
        if (result.status === 'success') {
            const logsContent = document.getElementById('ai-logs-content');
            if (logsContent) logsContent.textContent = '日志已清除。';
        }
    } catch (error: any) {
        showNotification({ message: `清除日志时发生网络错误: ${error.message}`, type: 'error' });
    }
}

// Delegated event listener for the settings page
document.addEventListener('click', async (e) => {
    const target = e.target as HTMLElement;

    // Modal Triggers
    const modalTrigger = target.closest('[data-modal-target]');
    if (modalTrigger) {
        const modalId = modalTrigger.getAttribute('data-modal-target');
        if (modalId) {
            const modal = document.getElementById(modalId);
            if (modal) {
                modal.classList.add('show');
                if (modalId === 'ai-logs-modal') {
                    fetchAndDisplayAILogs();
                }
            }
        }
    }

    // Modal Close Buttons
    const modalCloseBtn = target.closest('.modal-close-btn');
    if (modalCloseBtn) {
        modalCloseBtn.closest('.modal-container')?.classList.remove('show');
    }
    
    // Click outside Modal to close
    if (target.matches('.modal-container')) {
        target.classList.remove('show');
    }

    // Form buttons
    const btn = target.closest('.btn');
    if (btn) {
        const form = btn.closest('form');
        switch (btn.id) {
            case 'save-settings-btn':
                const settingsForm = document.getElementById('settings-form') as HTMLFormElement;
                if (settingsForm) saveSettings(settingsForm);
                break;
            case 'save-ai-btn':
            case 'save-imageapi-btn':
            case 'save-github-btn':
            case 'save-webdav-btn':
                if (form) saveSettings(form);
                break;
            case 'test-ai-btn':
                if (form) testConnection(form, 'ai');
                break;
            case 'test-imageapi-btn':
                if (form) testConnection(form, 'imageapi');
                break;
            case 'test-github-btn':
                if (form) testConnection(form, 'github');
                break;
            case 'test-webdav-btn':
                if (form) testConnection(form, 'webdav');
                break;
            case 'backup-github-now-btn':
                backupNow('github');
                break;
            case 'backup-webdav-now-btn':
                backupNow('webdav');
                break;
            case 'clear-ai-logs-btn':
                clearAILogs();
                break;
            case 'upload-btn':
                document.getElementById('backup-file')?.click();
                break;
        }
    }
});

document.addEventListener('change', async (e) => {
    const target = e.target as HTMLInputElement;
    if (target.id === 'backup-file' && target.files?.[0]) {
        const file = target.files[0];
        const formData = new FormData();
        formData.append('backup', file);

        if (file.name.endsWith('.zip')) {
            const password = prompt('请输入备份文件密码 (如果无密码请留空):');
            if (password === null) {
                target.value = '';
                return;
            }
            formData.append('password', password);
        }
        
        showNotification({ message: '正在上传并恢复...', type: 'info' });
        try {
            const response = await fetch(`${apiBaseUrl}/api/backup/upload`, {
                method: 'POST',
                credentials: 'include',
                body: formData,
            });
            const result = await response.json();
            showNotification({ message: result.message, type: result.status });
        } catch (error: any) {
            showNotification({ message: `上传失败: ${error.message}`, type: 'error' });
        } finally {
            target.value = '';
        }
    }
});