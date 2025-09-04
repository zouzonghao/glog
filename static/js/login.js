document.getElementById('login-form').addEventListener('submit', function(event) {
    event.preventDefault();
    const form = event.target;
    const formData = new FormData(form);

    fetch(form.action, {
        method: 'POST',
        body: new URLSearchParams(formData)
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            window.location.href = '/admin/';
        } else {
            showNotification(data.message, 'error');
        }
    })
    .catch(error => {
        console.error('登录请求失败:', error);
        showNotification('登录请求失败，请检查网络！', 'error');
    });
});