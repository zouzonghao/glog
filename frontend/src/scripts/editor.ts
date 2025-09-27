import { showNotification } from '../utils/notifications';

const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

function handleAICoverCheckbox() {
    const aiCoverCheckbox = document.getElementById('ai_cover') as HTMLInputElement;
    const promptContainer = document.getElementById('ai-cover-prompt-container');
    if (aiCoverCheckbox && promptContainer) {
        promptContainer.style.display = aiCoverCheckbox.checked ? 'block' : 'none';
    }
}

async function savePost(form: HTMLFormElement) {
    const saveBtn = document.getElementById('save-btn');
    if (saveBtn) saveBtn.textContent = '保存中...';
    (saveBtn as HTMLButtonElement).disabled = true;

    const formData = new FormData(form);
    const data: { [key: string]: any } = {};
    formData.forEach((value, key) => {
        data[key] = value;
    });

    // Handle checkboxes explicitly
    data.ai_summary = (form.querySelector('#ai_summary') as HTMLInputElement).checked;
    data.ai_cover = (form.querySelector('#ai_cover') as HTMLInputElement).checked;
    data.is_private = (form.querySelector('#is_private') as HTMLInputElement).checked;

    const postId = (form.querySelector('#post-id') as HTMLInputElement).value;
    const isNewPost = postId === '0';

    // Convert id to number before sending to backend
    if (data.id) {
        data.id = parseInt(data.id, 10);
    }

    // For new posts, always use POST. For existing, use PUT.
    // The backend handler for /api/posts can handle both creation and updates based on the presence of an ID in the body.
    const url = `${apiBaseUrl}/api/posts`;
    const method = 'POST';

    try {
        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(data),
        });

        const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || '保存失败');
        }

        const newPostId = result.post_id;
        const newSlug = result.slug;

        // If it was a new post, store notification and navigate to the new edit page.
        if (isNewPost) {
            sessionStorage.setItem('glog_notification', JSON.stringify({ message: result.message || '文章已创建！', type: 'success' }));
            
            if ((window as any).Astro && (window as any).Astro.navigate) {
                (window as any).Astro.navigate(`/admin/editor?id=${newPostId}`);
            } else {
                window.location.href = `/admin/editor?id=${newPostId}`;
            }
            return; // Stop execution to allow navigation to complete
        }

        // For existing posts, show notification directly as there is no navigation.
        showNotification({ message: result.message || '文章已更新！', type: 'success' });

        const openPostLink = document.querySelector('.open-post-link') as HTMLAnchorElement;
        if (openPostLink && newSlug) {
            openPostLink.href = `/post/${newSlug}`;
            openPostLink.style.display = 'inline-block';
        }

    } catch (error: any) {
        showNotification({ message: error.message, type: 'error' });
    } finally {
        if (saveBtn) saveBtn.textContent = '💾 保存文章';
        (saveBtn as HTMLButtonElement).disabled = false;
    }
}

function initEditorPage() {
    handleAICoverCheckbox(); // Initial check
}

// Use event delegation for clicks
document.addEventListener('click', (e) => {
    const target = e.target as HTMLElement;

    if (target.matches('#save-btn')) {
        const form = document.getElementById('app-form') as HTMLFormElement;
        if (form) {
            savePost(form);
        }
    }
});

// Use event delegation for changes
document.addEventListener('change', (e) => {
    const target = e.target as HTMLElement;

    if (target.matches('#ai_cover')) {
        handleAICoverCheckbox();
    }
});


// Run initial setup
initEditorPage();