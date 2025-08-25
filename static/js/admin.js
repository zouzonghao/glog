document.addEventListener('DOMContentLoaded', function() {
    const postListBody = document.querySelector('.post-list-body');

    postListBody.addEventListener('focusin', function(event) {
        if (event.target.classList.contains('delete-wrapper')) {
            const wrapper = event.target;
            const confirmButton = wrapper.querySelector('.delete-confirm');
            
            // Disable the button immediately
            confirmButton.classList.add('disabled');
            
            // Enable it after 1 second
            setTimeout(() => {
                confirmButton.classList.remove('disabled');
            }, 1000);
        }
    });

    postListBody.addEventListener('click', function(event) {
        const confirmButton = event.target;
        if (confirmButton.classList.contains('delete-confirm') && !confirmButton.classList.contains('disabled')) {
            const postId = confirmButton.dataset.id;
            
            fetch(`/admin/delete/${postId}`, {
                method: 'POST',
            })
            .then(response => response.json())
            .then(data => {
                if (data.status === 'success') {
                    showNotification(data.message, 'success');
                    const itemToRemove = confirmButton.closest('.post-list-item');
                    if (itemToRemove) {
                        itemToRemove.remove();
                    }
                } else {
                    showNotification(data.message, 'error');
                }
            })
            .catch(error => {
                console.error('删除失败:', error);
                showNotification('删除文章时出错！', 'error');
            });
        }
    });
});