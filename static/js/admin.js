document.addEventListener('DOMContentLoaded', function() {
    const postListBody = document.querySelector('.post-list-body');

    // --- 单个删除逻辑 ---
    postListBody.addEventListener('focusin', function(event) {
        if (event.target.classList.contains('delete-wrapper')) {
            const wrapper = event.target;
            const confirmButton = wrapper.querySelector('.delete-confirm');
            
            confirmButton.classList.add('disabled');
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

    // --- 批量操作逻辑 ---
    const selectAllCheckbox = document.getElementById('select-all-posts');
    const postCheckboxes = document.querySelectorAll('.post-checkbox');
    const batchDeleteBtn = document.getElementById('batch-delete-btn');
    const batchSetPrivateBtn = document.getElementById('batch-set-private-btn');
    const batchSetPublicBtn = document.getElementById('batch-set-public-btn');
    const modalContainer = document.getElementById('modal-container');
    const modalConfirmBtn = document.getElementById('modal-confirm-btn');
    const modalCancelBtn = document.getElementById('modal-cancel-btn');

    let currentAction = null;
    let currentIsPrivate = false;

    function updateBatchButtons() {
        const selectedIds = getSelectedPostIds();
        const hasSelection = selectedIds.length > 0;
        batchDeleteBtn.disabled = !hasSelection;
        batchSetPrivateBtn.disabled = !hasSelection;
        batchSetPublicBtn.disabled = !hasSelection;
    }

    function getSelectedPostIds() {
        return Array.from(postCheckboxes)
            .filter(checkbox => checkbox.checked)
            .map(checkbox => parseInt(checkbox.dataset.id, 10));
    }

    selectAllCheckbox.addEventListener('change', function() {
        postCheckboxes.forEach(checkbox => {
            checkbox.checked = selectAllCheckbox.checked;
        });
        updateBatchButtons();
    });

    postCheckboxes.forEach(checkbox => {
        checkbox.addEventListener('change', function() {
            if (!this.checked) {
                selectAllCheckbox.checked = false;
            } else {
                if (Array.from(postCheckboxes).every(cb => cb.checked)) {
                    selectAllCheckbox.checked = true;
                }
            }
            updateBatchButtons();
        });
    });

    async function handleBatchAction() {
        const ids = getSelectedPostIds();
        if (ids.length === 0) {
            showNotification('请至少选择一篇文章。', 'info');
            return;
        }

        try {
            const response = await fetch('/admin/posts/batch-update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ ids, action: currentAction, is_private: currentIsPrivate }),
            });
            const data = await response.json();

            if (data.status === 'success') {
                showNotification(data.message, 'success');
                // 为了确保数据一致性，在短暂显示通知后重新加载页面
                setTimeout(() => window.location.reload(), 1000);
            } else {
                showNotification(data.message, 'error');
            }
        } catch (error) {
            console.error('批量操作失败:', error);
            showNotification('批量操作时出错！', 'error');
        } finally {
            // 仅在删除操作时关闭模态框
            if (currentAction === 'delete') {
                closeModal();
            }
        }
    }

    function showModal() {
        modalContainer.style.display = 'flex';
    }

    function closeModal() {
        modalContainer.style.display = 'none';
    }

    batchDeleteBtn.addEventListener('click', () => {
        currentAction = 'delete';
        showModal();
    });
    batchSetPrivateBtn.addEventListener('click', () => {
        currentAction = 'set-private';
        currentIsPrivate = true;
        handleBatchAction();
    });
    batchSetPublicBtn.addEventListener('click', () => {
        currentAction = 'set-private';
        currentIsPrivate = false;
        handleBatchAction();
    });

    modalConfirmBtn.addEventListener('click', handleBatchAction);
    modalCancelBtn.addEventListener('click', closeModal);
    modalContainer.addEventListener('click', function(event) {
        if (event.target === modalContainer) {
            closeModal();
        }
    });
});