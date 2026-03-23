document.addEventListener('DOMContentLoaded', function() {
    const saveBtn = document.getElementById('save-btn');
    const contentArea = document.getElementById('content');
    const titleInput = document.getElementById('title');
    const postIdInput = document.getElementById('post-id');
    const openLink = document.querySelector('.editor-actions a.open-post-link');

    const updateButtonStates = () => {
        const isNewPost = postIdInput.value === '0';
        const isContentEmpty = contentArea.value.trim() === '';

        if (isNewPost) {
            saveBtn.textContent = '💾 保存文章';
            saveBtn.disabled = isContentEmpty;
            saveBtn.style.opacity = isContentEmpty ? '0.5' : '1';
            saveBtn.style.borderColor = '';
            saveBtn.style.color = '';
        } else {
            saveBtn.disabled = false;
            saveBtn.style.opacity = '1';
            if (isContentEmpty) {
                saveBtn.textContent = '🗑️ 删除文章';
                saveBtn.style.borderColor = '#cb2a42';
                saveBtn.style.color = '#cb2a42';
            } else {
                saveBtn.textContent = '💾 保存文章';
                saveBtn.style.borderColor = '';
                saveBtn.style.color = '';
            }
        }

        if (isNewPost) {
            openLink.classList.add('disabled');
        } else {
            openLink.classList.remove('disabled');
        }
    };

    contentArea.addEventListener('input', updateButtonStates);
    updateButtonStates();

    saveBtn.addEventListener('click', function(event) {
        event.preventDefault();

        const publishedAtInput = document.getElementById('published_at');
        const publishedAtValue = publishedAtInput.value;
        const dateTimeRegex = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/;

        if (!dateTimeRegex.test(publishedAtValue)) {
            showNotification('发布时间格式不正确，应为 YYYY-MM-DD HH:mm', 'error');
            return;
        }

        const form = document.getElementById('app-form');
        const formData = new FormData(form);

        fetch(form.action, {
            method: 'POST',
            body: new URLSearchParams(formData)
        })
        .then(response => response.json())
        .then(data => {
            let alertClass = 'info';
            if (data.status === 'success') {
                alertClass = 'success';
                if (postIdInput.value === '0' && data.post_id) {
                    postIdInput.value = data.post_id;
                    const newUrl = `/admin/editor?id=${data.post_id}`;
                    history.pushState({path: newUrl}, '', newUrl);
                }

                if (data.slug) {
                    openLink.href = `/post/${data.slug}`;
                }

                updateButtonStates();
            } else if (data.status === 'deleted') {
                alertClass = 'success';
                setTimeout(() => {
                    window.location.href = '/admin';
                }, 1500);
            } else if (data.status === 'error' || data.status === 'locked') {
                alertClass = 'error';
            }
            
            showNotification(data.message, alertClass);
        })
        .catch(error => {
            console.error('保存错误：', error);
            showNotification('保存时发生错误，请检查网络！', 'error');
        });
    });
});
