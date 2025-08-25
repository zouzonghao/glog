// AJAX form submission
document.getElementById('save-btn').addEventListener('click', function(event) {
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
    const postIdInput = document.getElementById('post-id');

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
            const openLink = document.querySelector('.editor-actions a.open-post-link');
            if (data.slug) {
                openLink.href = `/post/${data.slug}`;
            } else {
                openLink.href = '#';
            }
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

// Handle click on "Open Post" link
document.querySelector('.open-post-link').addEventListener('click', function(event) {
    const postId = document.getElementById('post-id').value;
    if (postId === '0') {
        event.preventDefault();
        showNotification('文章尚未保存，无法打开！', 'info');
    }
});