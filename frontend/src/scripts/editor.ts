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

    const url = isNewPost ? `${apiBaseUrl}/api/posts` : `${apiBaseUrl}/api/posts/${postId}`;
    const method = isNewPost ? 'POST' : 'PUT';

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

        showNotification({ message: '文章保存成功！', type: 'success' });
        
        const newPostId = result.data.id;
        const newSlug = result.data.slug;

        // Update form and URL for subsequent saves
        (document.getElementById('post-id') as HTMLInputElement).value = newPostId;
        const openPostLink = document.querySelector('.open-post-link') as HTMLAnchorElement;
        if (openPostLink) {
            openPostLink.href = `/post/${newSlug}`;
            openPostLink.style.display = 'inline-block'; // Show the link
        }
        
        // Update browser URL without a full reload
        history.pushState(null, '', `/admin/editor/${newPostId}`);
        
        // If it was a new post, navigate to the new edit page to fully reload state if needed
        if (isNewPost && (window as any).Astro) {
            (window as any).Astro.navigate(`/admin/editor/${newPostId}`);
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