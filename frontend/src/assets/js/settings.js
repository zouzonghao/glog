document.addEventListener('DOMContentLoaded', function() {
    // --- Main Settings Form ---
    const mainSettingsForm = document.getElementById('settings-form');
    if (mainSettingsForm) {
        const saveSettingsBtn = document.getElementById('save-settings-btn');
        saveSettingsBtn.addEventListener('click', function(event) {
            event.preventDefault();
            saveFormData(mainSettingsForm);
        });
    }

    // --- Modal Setup using Global Function ---
    setupGlobalModal('ai-modal', 'ai-settings-btn');
    setupGlobalModal('github-modal', 'github-backup-btn');
    setupGlobalModal('webdav-modal', 'webdav-backup-btn');
    setupGlobalModal('imageapi-modal', 'imageapi-settings-btn');
    setupGlobalModal('ai-logs-modal', 'ai-logs-btn');
    // Note: password-prompt-modal is now opened programmatically when needed.

    // --- AI Logs Button Logic ---
    const aiLogsBtn = document.getElementById('ai-logs-btn');
    if (aiLogsBtn) {
        aiLogsBtn.addEventListener('click', fetchAndDisplayAILogs);
    }

    const clearAiLogsBtn = document.getElementById('clear-ai-logs-btn');
    if (clearAiLogsBtn) {
        clearAiLogsBtn.addEventListener('click', clearAILogs);
    }

    // --- Form-specific Logic inside Modals ---
    attachModalFormLogic('save-ai-btn', 'ai-settings-form', 'ai-modal');
    attachModalFormLogic('save-imageapi-btn', 'imageapi-settings-form', 'imageapi-modal');
    attachModalFormLogic('save-github-btn', 'github-settings-form', 'github-modal');
    attachModalFormLogic('save-webdav-btn', 'webdav-settings-form', 'webdav-modal');

    attachTestConnectionLogic('test-ai-btn', 'ai-settings-form', '/admin/setting/test-ai');
    attachTestConnectionLogic('test-github-btn', 'github-settings-form', '/admin/setting/test-github');
    attachTestConnectionLogic('test-webdav-btn', 'webdav-settings-form', '/admin/setting/test-webdav');
    // Special logic for ImageAPI test button
    const testImageApiBtn = document.getElementById('test-imageapi-btn');
    if (testImageApiBtn) {
        testImageApiBtn.addEventListener('click', (e) => {
            e.preventDefault();
            const form = document.getElementById('imageapi-settings-form');
            const testButton = e.target;
            showNotification('测试中...', 'info');
            testButton.disabled = true;

            fetch('/admin/setting/test-imageapi', {
                method: 'POST',
                body: new URLSearchParams(new FormData(form))
            })
            .then(res => res.json())
            .then(data => {
                showNotification(data.message, data.status);
                if (data.status === 'success' && data.models && data.models.length > 0) {
                    updateImageApiModelSelector(data.models);
                }
            })
            .catch(err => {
                console.error('测试连接失败:', err);
                showNotification('测试请求失败，请检查网络或后台日志！', 'error');
            })
            .finally(() => testButton.disabled = false);
        });
    }

    attachBackupNowLogic('backup-github-now-btn', '/admin/setting/backup-github-now');
    attachBackupNowLogic('backup-webdav-now-btn', '/admin/setting/backup-webdav-now');


    // --- Backup and Upload Logic ---
    const uploadBtn = document.getElementById('upload-btn');
    const backupFile = document.getElementById('backup-file');
    if (uploadBtn && backupFile) {
        uploadBtn.addEventListener('click', function(event) {
            event.preventDefault();
            backupFile.click();
        });

        backupFile.addEventListener('change', async function(event) {
            const file = event.target.files[0];
            if (!file) return;

            if (file.name.endsWith('.zip')) {
                const password = await showGlobalPasswordPrompt('请输入备份文件密码：');
                if (password === null) {
                    event.target.value = '';
                    return;
                }
                uploadZipFile(file, password);
            } else if (file.name.endsWith('.json')) {
                handleJsonFile(file);
            } else {
                showNotification('请选择 .zip 或 .json 格式的备份文件。', 'error');
                event.target.value = '';
            }
        });
    }
});

function handleJsonFile(file) {
    const reader = new FileReader();
    reader.onload = function(e) {
        try {
            const jsonData = JSON.parse(e.target.result);
            if (validateBackupJson(jsonData)) {
                uploadJsonData(jsonData);
            } else {
                showNotification('JSON 文件结构不正确。必须包含 posts 数组。', 'error');
            }
        } catch (error) {
            showNotification('解析 JSON 文件失败: ' + error.message, 'error');
        } finally {
            document.getElementById('backup-file').value = '';
        }
    };
    reader.onerror = function() {
        showNotification('读取文件失败！', 'error');
        document.getElementById('backup-file').value = '';
    };
    reader.readAsText(file);
}

function validateBackupJson(data) {
    if (typeof data !== 'object' || data === null) return false;
    if (!data.hasOwnProperty('posts') || !Array.isArray(data.posts)) return false;
    if (data.hasOwnProperty('settings') && (typeof data.settings !== 'object' || data.settings === null)) return false;
    
    // Optional: Check a sample post structure
    if (data.posts.length > 0) {
        const samplePost = data.posts[0];
        if (typeof samplePost.title === 'undefined' || 
            typeof samplePost.content === 'undefined' ||
            typeof samplePost.is_private === 'undefined' ||
            typeof samplePost.published_at === 'undefined') {
            return false;
        }
    }
    return true;
}

function uploadJsonData(jsonData) {
    showNotification('正在上传并恢复...', 'info');
    fetch('/admin/setting/upload', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(jsonData)
    })
    .then(response => response.json())
    .then(data => {
        showNotification(data.message, data.status);
    })
    .catch(error => {
        console.error('上传错误:', error);
        showNotification('上传失败，请检查网络或后台日志！', 'error');
    });
}

function uploadZipFile(file, password = '') {
    const formData = new FormData();
    formData.append('backup', file);
    if (password) {
        formData.append('password', password);
    }

    showNotification('正在上传并恢复...', 'info');

    fetch('/admin/setting/upload', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        showNotification(data.message, data.status);
        document.getElementById('backup-file').value = '';
    })
    .catch(error => {
        console.error('上传错误:', error);
        showNotification('上传失败，请检查网络或后台日志！', 'error');
        document.getElementById('backup-file').value = '';
    });
}

function attachModalFormLogic(saveBtnId, formId, modalId) {
    const saveBtn = document.getElementById(saveBtnId);
    const form = document.getElementById(formId);
    const modal = document.getElementById(modalId);
    if (saveBtn && form && modal) {
        saveBtn.addEventListener('click', (e) => {
            e.preventDefault();
            saveFormData(form, () => {
                modal.classList.remove('show');
            });
        });
    }
}

function attachTestConnectionLogic(testBtnId, formId, testUrl) {
    const testBtn = document.getElementById(testBtnId);
    const form = document.getElementById(formId);
    if (testBtn && form) {
        testBtn.addEventListener('click', (e) => {
            e.preventDefault();
            const testButton = e.target;
            showNotification('测试中...', 'info');
            testButton.disabled = true;

            fetch(testUrl, {
                method: 'POST',
                body: new URLSearchParams(new FormData(form))
            })
            .then(res => res.json())
            .then(data => showNotification(data.message, data.status))
            .catch(err => {
                console.error('测试连接失败:', err);
                showNotification('测试请求失败，请检查网络或后台日志！', 'error');
            })
            .finally(() => testButton.disabled = false);
        });
    }
}

function attachBackupNowLogic(backupNowBtnId, backupNowUrl) {
    const backupNowBtn = document.getElementById(backupNowBtnId);
    if (backupNowBtn) {
        backupNowBtn.addEventListener('click', (e) => {
            e.preventDefault();
            const backupButton = e.target;
            showNotification('正在备份...', 'info');
            backupButton.disabled = true;

            fetch(backupNowUrl, {
                method: 'POST'
            })
            .then(res => res.json())
            .then(data => showNotification(data.message, data.status))
            .catch(err => {
                console.error('立即备份失败:', err);
                showNotification('备份请求失败，请检查网络或后台日志！', 'error');
            })
            .finally(() => backupButton.disabled = false);
        });
    }
}

function saveFormData(formElement, callback) {
    const formData = new FormData(formElement);
    fetch('/admin/setting', {
        method: 'POST',
        body: new URLSearchParams(formData)
    })
    .then(response => response.json())
    .then(data => {
        showNotification(data.message, data.status);
        if (data.status === 'success') {
            formElement.querySelectorAll('input[type="password"]').forEach(input => input.value = '');
            if (callback) callback();
        }
    })
    .catch(error => {
        console.error('表单提交错误：', error);
        showNotification('保存时发生错误，请检查网络连接！', 'error');
    });
}

function fetchAndDisplayAILogs() {
    const logsContent = document.getElementById('ai-logs-content');
    if (!logsContent) return;

    logsContent.textContent = '正在加载日志...';

    fetch('/admin/setting/ai-logs')
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                logsContent.textContent = data.logs || '暂无 AI 日志。';
            } else {
                logsContent.textContent = '加载日志失败: ' + data.message;
            }
        })
        .catch(error => {
            console.error('获取AI日志失败:', error);
            logsContent.textContent = '加载日志时发生网络错误。';
        });
}

function updateImageApiModelSelector(providers) {
    const container = document.getElementById('imageapi-model-container');
    if (!container) return;

    const currentModel = container.querySelector('input[name="imageapi_model"]')?.value;

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

    // Replace the input with the new select element
    const label = container.querySelector('label');
    container.innerHTML = ''; // Clear the container
    container.appendChild(label);
    container.appendChild(select);
}

function clearAILogs() {
    showNotification('正在清除日志...', 'info');
    fetch('/admin/setting/ai-logs/clear', {
        method: 'POST'
    })
    .then(response => response.json())
    .then(data => {
        showNotification(data.message, data.status);
        if (data.status === 'success') {
            // Also clear the content in the modal
            const logsContent = document.getElementById('ai-logs-content');
            if (logsContent) {
                logsContent.textContent = '日志已清除。';
            }
        }
    })
    .catch(error => {
        console.error('清除AI日志失败:', error);
        showNotification('清除日志时发生网络错误。', 'error');
    });
}