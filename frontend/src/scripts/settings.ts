import { showNotification } from '../utils/notifications';

// This script runs on the client, so we can safely access document and window.
const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

// General site settings
const saveSettingsBtn = document.getElementById('save-settings-btn');
const settingsForm = document.getElementById('settings-form') as HTMLFormElement;

if (saveSettingsBtn && settingsForm) {
    saveSettingsBtn.addEventListener('click', async () => {
        const formData = new FormData(settingsForm);
        
        // Manually handle password field to not send it if empty
        const data: { [key: string]: any } = {};
        formData.forEach((value, key) => {
            data[key] = value;
        });

        if (!data.password) {
            delete data.password;
        }

        try {
            const response = await fetch(`${apiBaseUrl}/api/admin/setting`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            });

            const result = await response.json();
            if (response.ok) {
                showNotification({ message: '站点信息保存成功！', type: 'success' });
            } else {
                throw new Error(result.error || '保存失败');
            }
        } catch (error: any) {
            showNotification({ message: error.message, type: 'error' });
        }
    });
}

// AI settings
const saveAiBtn = document.getElementById('save-ai-btn');
if (saveAiBtn) {
    saveAiBtn.addEventListener('click', async () => {
        // In a real implementation, you would gather form data and send it.
        showNotification({ message: 'AI 设置已保存 (模拟)', type: 'success' });
    });
}

// Add event listeners for other buttons (ImageAPI, GitHub, WebDAV, etc.) here.