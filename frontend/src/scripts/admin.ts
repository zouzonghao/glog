import { showNotification } from '../utils/notifications';

document.addEventListener('DOMContentLoaded', () => {
    const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';
    const selectAllCheckbox = document.getElementById('select-all-posts') as HTMLInputElement;
    const postCheckboxes = document.querySelectorAll('.post-checkbox') as NodeListOf<HTMLInputElement>;
    const batchDeleteBtn = document.getElementById('batch-delete-btn') as HTMLButtonElement;
    const batchSetPrivateBtn = document.getElementById('batch-set-private-btn') as HTMLButtonElement;
    const batchSetPublicBtn = document.getElementById('batch-set-public-btn') as HTMLButtonElement;

    function updateBatchButtons() {
        const selectedIds = getSelectedPostIds();
        const hasSelection = selectedIds.length > 0;
        batchDeleteBtn.disabled = !hasSelection;
        batchSetPrivateBtn.disabled = !hasSelection;
        batchSetPublicBtn.disabled = !hasSelection;
    }

    function getSelectedPostIds() {
        return Array.from(postCheckboxes)
            .filter(cb => cb.checked)
            .map(cb => cb.dataset.id);
    }

    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', () => {
            postCheckboxes.forEach(checkbox => {
                checkbox.checked = selectAllCheckbox.checked;
            });
            updateBatchButtons();
        });
    }

    postCheckboxes.forEach(checkbox => {
        checkbox.addEventListener('change', () => {
            if (!checkbox.checked) {
                selectAllCheckbox.checked = false;
            }
            updateBatchButtons();
        });
    });

    // Single delete logic
    document.querySelectorAll('.delete-wrapper').forEach(wrapper => {
        const confirmSpan = wrapper.querySelector('.delete-confirm') as HTMLElement;
        confirmSpan.addEventListener('click', async () => {
            const postId = confirmSpan.dataset.id;
            if (confirm('确定要删除这篇文章吗？')) {
                // API call to delete post
                showNotification({ message: `删除了文章 #${postId} (模拟)`, type: 'success' });
            }
        });
    });

    // Initial state
    updateBatchButtons();
});