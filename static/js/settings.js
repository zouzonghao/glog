// AJAX for saving settings
document.getElementById('save-settings-btn').addEventListener('click', function(event) {
    event.preventDefault();

    const form = document.getElementById('settings-form');
    const formData = new FormData(form);

    fetch(form.action, {
        method: 'POST',
        body: new URLSearchParams(formData)
    })
    .then(response => response.json())
    .then(data => {
        showNotification(data.message, data.status);

        if (data.status === 'success') {
            document.getElementById('password').value = '';
            document.getElementById('openai_token').value = '';
        }
    })
    .catch(error => {
        console.error('表单提交错误：', error);
        showNotification('保存时发生错误，请检查网络连接！', 'error');
    });
});

// AJAX for testing AI connection
document.getElementById('test-ai-btn').addEventListener('click', function(event) {
    event.preventDefault();
    const baseURL = document.getElementById('openai_base_url').value;
    const token = document.getElementById('openai_token').value;
    const model = document.getElementById('openai_model').value;
    const testBtn = this;

    showNotification('测试中...', 'info');
    testBtn.style.pointerEvents = 'none'; // Disable link
    testBtn.style.opacity = '0.5';

    const formData = new URLSearchParams();
    formData.append('openai_base_url', baseURL);
    formData.append('openai_token', token);
    formData.append('openai_model', model);

    fetch('/settings/test-ai', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        showNotification(data.message, data.status);
    })
    .catch(error => {
        console.error('AI 测试错误：', error);
        showNotification('测试请求失败，请检查网络或后台日志！', 'error');
    })
    .finally(() => {
        testBtn.style.pointerEvents = 'auto'; // Re-enable link
        testBtn.style.opacity = '1';
    });
});

// Backup and Upload logic
document.getElementById('upload-btn').addEventListener('click', function(event) {
    event.preventDefault();
    document.getElementById('backup-file').click();
});

document.getElementById('backup-file').addEventListener('change', function(event) {
    const file = event.target.files[0];
    if (!file) {
        return;
    }

    const uploadFile = (file) => {
        const formData = new FormData();
        formData.append('backup', file);

        showNotification('正在上传并恢复...', 'info');

        fetch('/settings/upload', {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            showNotification(data.message, data.status);
            event.target.value = ''; // Reset file input
        })
        .catch(error => {
            console.error('上传错误:', error);
            showNotification('上传失败，请检查网络或后台日志！', 'error');
        });
    };

    if (file.name.endsWith('.zip')) {
        uploadFile(file);
    } else if (file.name.endsWith('.json')) {
        const reader = new FileReader();
        reader.onload = function(e) {
            try {
                const data = JSON.parse(e.target.result);
                if (Array.isArray(data) && (data.length === 0 || (data[0].hasOwnProperty('title') && data[0].hasOwnProperty('content')))) {
                    uploadFile(file);
                } else {
                    showNotification('JSON 文件结构无效。应为一个数组，且包含 title 和 content 字段。', 'error');
                    event.target.value = '';
                }
            } catch (error) {
                showNotification('解析 JSON 文件失败，请检查文件格式。', 'error');
                event.target.value = '';
            }
        };
        reader.readAsText(file);
    } else {
        showNotification('请选择一个 .zip 或 .json 格式的备份文件。', 'error');
        event.target.value = '';
    }
});