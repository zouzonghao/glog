import { showNotification } from '../utils/notifications';

document.addEventListener('DOMContentLoaded', () => {
    const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';
    const form = document.getElementById('app-form') as HTMLFormElement;
    const saveBtn = document.getElementById('save-btn');
    const openPostLink = document.querySelector('.open-post-link') as HTMLAnchorElement;

    if (saveBtn && form) {
        saveBtn.addEventListener('click', async () => {
            const formData = new FormData(form);
            const data: { [key: string]: any } = {};
            formData.forEach((value, key) => {
                // Handle checkbox values
                if (value === 'on') {
                    data[key] = true;
                } else {
                    data[key] = value;
                }
            });
            
            // Ensure checkboxes that are not checked are sent as false
            ['ai_summary', 'ai_cover', 'is_private'].forEach(key => {
                if (!data[key]) {
                    data[key] = false;
                }
            });

            try {
                const response = await fetch(`${apiBaseUrl}/api/admin/save`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data),
                });

                const result = await response.json();
                if (response.ok && result.code === 0) {
                    showNotification({ message: '文章保存成功！', type: 'success' });
                    // Update the URL and post ID for future saves
                    const newPostId = result.data.id;
                    const newSlug = result.data.slug;
                    (document.getElementById('post-id') as HTMLInputElement).value = newPostId;
                    if (openPostLink) {
                        openPostLink.href = `/post/${newSlug}`;
                    }
                    // Update browser URL without reloading
                    history.pushState(null, '', `/admin/editor?id=${newPostId}`);
                } else {
                    throw new Error(result.error || '保存失败');
                }
            } catch (error: any) {
                showNotification({ message: error.message, type: 'error' });
            }
        });
    }
});