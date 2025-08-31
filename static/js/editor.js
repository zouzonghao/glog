document.addEventListener('DOMContentLoaded', function() {
    const saveBtn = document.getElementById('save-btn');
    const contentArea = document.getElementById('content');
    const postIdInput = document.getElementById('post-id');
    const openLink = document.querySelector('.editor-actions a.open-post-link');

    // Function to update the state of all action buttons
    const updateButtonStates = () => {
        const isNewPost = postIdInput.value === '0';
        const isContentEmpty = contentArea.value.trim() === '';

        // --- Update Save Button ---
        if (isNewPost) {
            saveBtn.textContent = '💾 保存文章';
            saveBtn.disabled = isContentEmpty;
            saveBtn.style.opacity = isContentEmpty ? '0.5' : '1';
            saveBtn.style.borderColor = '';
            saveBtn.style.color = '';
        } else { // Editing existing post
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

        // --- Update Open Post Link ---
        if (isNewPost) {
            openLink.classList.add('disabled');
        } else {
            openLink.classList.remove('disabled');
        }
    };

    // Add event listener for content changes
    contentArea.addEventListener('input', updateButtonStates);

    // Initial check on page load
    updateButtonStates();

    // AJAX form submission
    saveBtn.addEventListener('click', function(event) {
        event.preventDefault();

        // Validate the published_at time format
        const publishedAtInput = document.getElementById('published_at');
        const publishedAtValue = publishedAtInput.value;
        const dateTimeRegex = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/;

        if (!dateTimeRegex.test(publishedAtValue)) {
            showNotification('发布时间格式不正确，应为 YYYY-MM-DD HH:mm', 'error');
            return; // Stop the submission
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
                // Update post ID for new posts
                if (postIdInput.value === '0' && data.post_id) {
                    postIdInput.value = data.post_id;
                    // Update browser URL to reflect the new post ID for editing
                    const newUrl = `/admin/editor?id=${data.post_id}`;
                    history.pushState({path: newUrl}, '', newUrl);
                }

                // Dynamically update "Open Post" link
                if (data.slug) {
                    openLink.href = `/post/${data.slug}`;
                }
                updateButtonStates(); // Re-check all button states
                
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

    // Add a disabled class to the CSS to handle the link
    const style = document.createElement('style');
    style.innerHTML = `
        .btn.disabled {
            opacity: 0.5;
            cursor: not-allowed;
            pointer-events: none;
        }
    `;
    document.head.appendChild(style);
});