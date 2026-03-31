document.addEventListener('DOMContentLoaded', function() {
    const mainSettingsForm = document.getElementById('settings-form');
    if (mainSettingsForm) {
        const saveSettingsBtn = document.getElementById('save-settings-btn');
        saveSettingsBtn.addEventListener('click', function(event) {
            event.preventDefault();
            saveFormData(mainSettingsForm);
        });
    }

    setupGlobalModal('sync-modal', 'sync-btn');

    attachModalFormLogic('save-sync-btn', 'sync-settings-form', 'sync-modal');

    attachSyncTestLogic('test-sync-btn');

    attachSyncNowLogic('sync-now-btn');

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
            saveFormData(form, (data) => {
                if (data.status === 'success') {
                    modal.classList.remove('show');
                }
            });
        });
    }
}

function saveFormData(formElement, callback, options = {}) {
    const shouldNotify = options.notify !== false;
    const formData = new FormData(formElement);
    fetch('/admin/setting', {
        method: 'POST',
        body: new URLSearchParams(formData)
    })
    .then(response => response.json())
    .then(data => {
        if (shouldNotify) {
            showNotification(data.message, data.status);
        }
        if (data.status === 'success') {
            formElement.querySelectorAll('input[type="password"]').forEach(input => input.value = '');
        }
        if (callback) callback(data);
    })
    .catch(error => {
        console.error('表单提交错误：', error);
        const fallback = { status: 'error', message: '保存时发生错误，请检查网络连接！' };
        if (shouldNotify) {
            showNotification(fallback.message, 'error');
        }
        if (callback) callback(fallback);
    });
}

function attachSyncTestLogic(testBtnId) {
    const testBtn = document.getElementById(testBtnId);
    const form = document.getElementById('sync-settings-form');
    if (testBtn && form) {
        testBtn.addEventListener('click', (e) => {
            e.preventDefault();
            const testButton = e.target;
            showNotification('测试中...', 'info');
            testButton.disabled = true;

            fetch('/admin/setting/status', {
                method: 'POST',
                body: new URLSearchParams(new FormData(form))
            })
            .then(res => res.json())
            .then(data => {
                showNotification(data.message, data.status);
                if (data.status === 'success' && data.data) {
                    updateSyncStatusDisplay(data.data);
                }
            })
            .catch(err => {
                console.error('测试连接失败:', err);
                showNotification('测试请求失败，请检查网络或后台日志！', 'error');
            })
            .finally(() => testButton.disabled = false);
        });
    }
}

function formatDateTime(isoString) {
    if (!isoString || isoString === '0001-01-01T00:00:00Z') {
        return '';
    }
    const date = new Date(isoString);
    if (isNaN(date.getTime())) {
        return isoString;
    }
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
    });
}

function updateSyncStatusDisplay(statusData) {
    const statusInfo = document.getElementById('sync-status-info');
    const statusText = document.getElementById('sync-status-text');
    
    if (statusInfo && statusText && statusData) {
        statusInfo.style.display = 'block';
        
        let statusHtml = '';
        if (!statusData.configured) {
            statusHtml = '<span style="color: #999;">未配置 WebDAV</span>';
        } else {
            const statusMap = {
                'synced': '<span style="color: #4CAF50;">✓ 已同步</span>',
                'local_ahead': '<span style="color: #FF9800;">↑ 本地有更新</span>',
                'remote_ahead': '<span style="color: #2196F3;">↓ 远程有更新</span>',
                'no_remote': '<span style="color: #999;">远程无数据</span>'
            };
            statusHtml = statusMap[statusData.status] || statusData.status;
            const localTime = formatDateTime(statusData.local_modified_at);
            const remoteTime = formatDateTime(statusData.remote_modified_at);
            if (localTime) {
                statusHtml += `<br><small>本地更新: ${localTime}</small>`;
            }
            if (remoteTime) {
                statusHtml += `<br><small>远程更新: ${remoteTime}</small>`;
            }
        }
        statusText.innerHTML = statusHtml;
    }
}

function attachSyncNowLogic(syncNowBtnId) {
    const syncNowBtn = document.getElementById(syncNowBtnId);
    if (syncNowBtn) {
        syncNowBtn.addEventListener('click', (e) => {
            e.preventDefault();
            const syncButton = e.target;
            showNotification('正在同步...', 'info');
            syncButton.disabled = true;

            fetch('/admin/setting/sync', {
                method: 'POST'
            })
            .then(res => res.json())
            .then(data => {
                showNotification(data.message, data.status);
                loadSyncStatus();
            })
            .catch(err => {
                console.error('同步失败:', err);
                showNotification('同步请求失败，请检查网络或后台日志！', 'error');
            })
            .finally(() => syncButton.disabled = false);
        });
    }
}

function loadSyncStatus() {
    const form = document.getElementById('sync-settings-form');
    if (!form) return;
    
    fetch('/admin/setting/status', {
        method: 'POST',
        body: new URLSearchParams(new FormData(form))
    })
        .then(res => res.json())
        .then(data => {
            if (data.status === 'success' && data.data) {
                updateSyncStatusDisplay(data.data);
            }
        })
        .catch(err => {
            console.error('获取同步状态失败:', err);
        });
}
