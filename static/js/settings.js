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
    setupGlobalModal('image-style-modal', 'image-style-settings-btn');
    // Note: password-prompt-modal is now opened programmatically when needed.

    // --- Form-specific Logic inside Modals ---
    attachModalFormLogic('save-ai-btn', 'ai-settings-form', 'ai-modal');
    attachModalFormLogic('save-imageapi-btn', 'imageapi-settings-form', 'imageapi-modal');
    attachModalFormLogic('save-image-style-btn', 'image-style-settings-form', 'image-style-modal');
    attachImageStyleResetLogic();
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
            saveFormData(form, (data) => {
                if (data.status === 'success') {
                    modal.classList.remove('show');
                }
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

function attachImageStyleResetLogic() {
    const resetBtn = document.getElementById('reset-image-style-btn');
    const form = document.getElementById('image-style-settings-form');
    const modal = document.getElementById('image-style-modal');
    if (!resetBtn || !form || !modal) {
        console.error('图片风格恢复默认初始化失败：缺少必要 DOM 节点。');
        return;
    }

    resetBtn.addEventListener('click', (e) => {
        e.preventDefault();
        const confirmed = window.confirm('确定要恢复图片风格和提示词模板为默认值吗？');
        if (!confirmed) {
            return;
        }

        const styleInput = document.getElementById('imageapi_style');
        const postTemplateInput = document.getElementById('image_prompt_template_by_post');
        const hintTemplateInput = document.getElementById('image_prompt_template_by_hint');
        const styleDefaultInput = document.getElementById('imageapi_style_default');
        const postTemplateDefault = document.getElementById('image_prompt_template_by_post_default');
        const hintTemplateDefault = document.getElementById('image_prompt_template_by_hint_default');

        const requiredElements = {
            imageapi_style: styleInput,
            image_prompt_template_by_post: postTemplateInput,
            image_prompt_template_by_hint: hintTemplateInput,
            imageapi_style_default: styleDefaultInput,
            image_prompt_template_by_post_default: postTemplateDefault,
            image_prompt_template_by_hint_default: hintTemplateDefault
        };
        const missingIds = Object.keys(requiredElements).filter((id) => !requiredElements[id]);
        if (missingIds.length > 0) {
            console.error('恢复默认失败，缺少 DOM 节点:', missingIds.join(', '));
            showNotification('恢复默认失败：页面结构异常，请刷新后重试。', 'error');
            return;
        }

        styleInput.value = styleDefaultInput.value || '';
        postTemplateInput.value = postTemplateDefault.value || '';
        hintTemplateInput.value = hintTemplateDefault.value || '';

        showNotification('已恢复默认，正在保存...', 'info');
        saveFormData(form, (data) => {
            if (data.status === 'success') {
                showNotification('默认配置已恢复并保存。', 'success');
                modal.classList.remove('show');
                return;
            }
            showNotification(data.message || '恢复默认保存失败，请稍后重试。', 'error');
        }, { notify: false });
    });
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
