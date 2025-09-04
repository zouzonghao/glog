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

    // --- Modal Setup ---
    setupModal(
        'ai-modal', 
        'ai-settings-btn', 
        'save-ai-btn', 
        'test-ai-btn', 
        'ai-settings-form', 
        '/admin/setting/test-ai'
    );
    
    setupModal(
        'github-modal', 
        'github-backup-btn', 
        'save-github-btn', 
        'test-github-btn', 
        'github-settings-form', 
        '/admin/setting/test-github', 
        'backup-github-now-btn', 
        '/admin/setting/backup-github-now'
    );

    setupModal(
        'webdav-modal', 
        'webdav-backup-btn', 
        'save-webdav-btn', 
        'test-webdav-btn', 
        'webdav-settings-form', 
        '/admin/setting/test-webdav', 
        'backup-webdav-now-btn', 
        '/admin/setting/backup-webdav-now'
    );

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
                const password = await showPasswordPrompt('请输入备份文件密码：');
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

function setupModal(modalId, openBtnId, saveBtnId, testBtnId, formId, testUrl, backupNowBtnId, backupNowUrl) {
    const modal = document.getElementById(modalId);
    const openBtn = document.getElementById(openBtnId);
    if (!modal || !openBtn) return;

    const closeBtn = modal.querySelector('.modal-close-btn');
    const saveBtn = document.getElementById(saveBtnId);
    const testBtn = testBtnId ? document.getElementById(testBtnId) : null;
    const backupNowBtn = backupNowBtnId ? document.getElementById(backupNowBtnId) : null;
    const form = document.getElementById(formId);

    openBtn.addEventListener('click', () => modal.classList.add('show'));
    closeBtn.addEventListener('click', () => modal.classList.remove('show'));
    window.addEventListener('click', (event) => {
        if (event.target === modal) {
            modal.classList.remove('show');
        }
    });

    if (saveBtn) {
        saveBtn.addEventListener('click', (e) => {
            e.preventDefault();
            saveFormData(form, () => {
                modal.classList.remove('show');
            });
        });
    }

    if (testBtn) {
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

function showPasswordPrompt(title) {
    return new Promise((resolve) => {
        const modal = document.getElementById('password-prompt-modal');
        const form = modal.querySelector('form');
        const titleEl = document.getElementById('password-prompt-title');
        const inputEl = document.getElementById('password-prompt-input');
        const confirmBtn = document.getElementById('password-prompt-confirm-btn');
        const cancelBtn = document.getElementById('password-prompt-cancel-btn');
        const closeBtn = modal.querySelector('.modal-close-btn');

        titleEl.textContent = title;
        inputEl.value = '';

        const closeModal = (value) => {
            modal.classList.remove('show');
            confirmBtn.onclick = null;
            cancelBtn.onclick = null;
            closeBtn.onclick = null;
            form.onsubmit = null;
            window.removeEventListener('click', outsideClickListener);
            resolve(value);
        };

        const outsideClickListener = (event) => {
            if (event.target === modal) {
                closeModal(null);
            }
        };

        const submitHandler = (event) => {
            event.preventDefault();
            closeModal(inputEl.value);
        };

        confirmBtn.onclick = () => submitHandler(new Event('submit'));
        cancelBtn.onclick = () => closeModal(null);
        closeBtn.onclick = () => closeModal(null);
        form.onsubmit = submitHandler;
        window.addEventListener('click', outsideClickListener);

        modal.classList.add('show');
        inputEl.focus();
    });
}