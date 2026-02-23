document.addEventListener('DOMContentLoaded', function() {
    const saveBtn = document.getElementById('save-btn');
    const contentArea = document.getElementById('content');
    const titleInput = document.getElementById('title');
    const postIdInput = document.getElementById('post-id');
    const openLink = document.querySelector('.editor-actions a.open-post-link');
    const aiCoverCheckbox = document.getElementById('ai_cover');
    const aiCoverPromptContainer = document.getElementById('ai-cover-prompt-container');
    const aiCoverPromptInput = document.getElementById('ai_cover_prompt');
    const aiCoverPreviewContainer = document.getElementById('ai-cover-preview-container');
    const aiCoverPreviewImage = document.getElementById('ai-cover-preview-image');
    const aiCoverLoading = document.getElementById('ai-cover-loading');
    const generateCoverLink = document.getElementById('generate-cover-link');
    const aiModal = document.getElementById('editor-ai-modal');
    const aiModalMessage = document.getElementById('editor-ai-modal-message');
    const aiModalCloseBtn = aiModal ? aiModal.querySelector('.modal-close-btn') : null;
    const aiModalOkBtn = document.getElementById('editor-ai-modal-ok');
    
    let generateAbortController = null;
    let isPersistedCoverGenerating = false;

    if (document.getElementById('is-cover-generating')) {
        isPersistedCoverGenerating = document.getElementById('is-cover-generating').value === '1';
    }

    const isValidHttpUrl = (value) => {
        const trimmed = (value || '').trim();
        if (!trimmed) {
            return false;
        }

        try {
            const parsed = new URL(trimmed);
            return parsed.protocol === 'http:' || parsed.protocol === 'https:';
        } catch (error) {
            return false;
        }
    };

    const hideAiModal = () => {
        if (aiModal) {
            aiModal.classList.remove('show');
        }
    };

    const showAiModal = (message) => {
        if (!aiModal || !aiModalMessage) {
            showNotification(message, 'error');
            return;
        }

        aiModalMessage.textContent = message;
        aiModal.classList.add('show');
    };

    if (aiModalCloseBtn) {
        aiModalCloseBtn.addEventListener('click', hideAiModal);
    }
    if (aiModalOkBtn) {
        aiModalOkBtn.addEventListener('click', hideAiModal);
    }
    if (aiModal) {
        aiModal.addEventListener('click', (event) => {
            if (event.target === aiModal) {
                hideAiModal();
            }
        });
    }

    const updateAiCoverUI = () => {
        if (!aiCoverCheckbox || !aiCoverPromptContainer || !aiCoverPromptInput || !generateCoverLink || !aiCoverPreviewContainer || !aiCoverPreviewImage || !aiCoverLoading) {
            return;
        }

        if (!aiCoverCheckbox.checked) {
            aiCoverPromptContainer.style.display = 'none';
            generateCoverLink.style.display = 'none';
            aiCoverPreviewContainer.style.display = 'none';
            aiCoverPreviewImage.removeAttribute('src');
            aiCoverPreviewImage.style.display = 'none';
            aiCoverLoading.style.display = 'none';
            return;
        }

        aiCoverPromptContainer.style.display = 'block';
        const promptValue = aiCoverPromptInput.value.trim();

        if (isPersistedCoverGenerating) {
            aiCoverPreviewContainer.style.display = 'block';
            aiCoverPreviewImage.removeAttribute('src');
            aiCoverPreviewImage.style.display = 'none';
            aiCoverLoading.style.display = 'flex';
            generateCoverLink.style.display = 'none';
            return;
        }

        if (isValidHttpUrl(promptValue)) {
            aiCoverPreviewImage.src = promptValue;
            aiCoverPreviewContainer.style.display = 'block';
            aiCoverPreviewImage.style.display = 'block';
            aiCoverLoading.style.display = 'none';
            generateCoverLink.style.display = 'none';
            return;
        }

        aiCoverPreviewContainer.style.display = 'none';
        aiCoverPreviewImage.removeAttribute('src');
        aiCoverPreviewImage.style.display = 'none';
        aiCoverLoading.style.display = 'none';
        generateCoverLink.style.display = 'inline-block';
    };

    const updateGenerateButtonState = () => {
        if (!generateCoverLink) return;

        if (generateAbortController) {
            generateCoverLink.textContent = '🛑 取消生成';
            if (aiCoverPromptInput) aiCoverPromptInput.disabled = true;
        } else {
            generateCoverLink.textContent = '✨ 生成图片';
            if (aiCoverPromptInput) aiCoverPromptInput.disabled = false;
        }
        updateButtonStates();
    };

    if (aiCoverCheckbox) {
        aiCoverCheckbox.addEventListener('change', () => {
            updateAiCoverUI();
            // 如果用户关闭了 AI 封面，且正在生成中，则自动取消生成
            if (!aiCoverCheckbox.checked && generateAbortController) {
                generateAbortController.abort();
                generateAbortController = null;
                updateGenerateButtonState();
                showNotification('因关闭 AI 封面，生成已取消', 'info');
            }

            if (!aiCoverCheckbox.checked) {
                isPersistedCoverGenerating = false;
            }
        });
    }

    if (aiCoverPromptInput) {
        aiCoverPromptInput.addEventListener('input', () => {
            if (isPersistedCoverGenerating) {
                isPersistedCoverGenerating = false;
            }
            updateAiCoverUI();
        });
        aiCoverPromptInput.addEventListener('change', () => {
            if (isPersistedCoverGenerating) {
                isPersistedCoverGenerating = false;
            }
            updateAiCoverUI();
        });
    }

    if (generateCoverLink) {
        generateCoverLink.addEventListener('click', async function(event) {
            event.preventDefault();

            // 如果正在生成，点击则取消
            if (generateAbortController) {
                generateAbortController.abort();
                generateAbortController = null;
                updateGenerateButtonState();
                showNotification('已取消封面生成', 'info');
                return;
            }

            if (!aiCoverCheckbox.checked) {
                return;
            }

            const content = contentArea.value.trim();
            const promptValue = aiCoverPromptInput ? aiCoverPromptInput.value.trim() : '';
            if (!content && !promptValue) {
                showAiModal('请先填写文章内容，或输入自定义提示词，再生成 AI 封面。');
                return;
            }

            // 开始生成
            generateAbortController = new AbortController();
            updateGenerateButtonState();

            try {
                const formBody = new URLSearchParams({
                    id: postIdInput ? postIdInput.value : '0',
                    title: titleInput ? titleInput.value : '',
                    content: contentArea.value,
                    ai_cover_prompt: aiCoverPromptInput ? aiCoverPromptInput.value : ''
                });

                const response = await fetch('/admin/cover/generate', {
                    method: 'POST',
                    body: formBody,
                    signal: generateAbortController.signal
                });
                const data = await response.json();

                if (data.status !== 'success' || !data.cover) {
                    showAiModal(data.message || 'AI 封面生成失败，请稍后再试。');
                    return;
                }

                aiCoverPromptInput.value = data.cover;
                isPersistedCoverGenerating = false;
                updateAiCoverUI();
                showNotification('AI 封面生成成功，请保存文章。', 'success');
            } catch (error) {
                if (error.name === 'AbortError') {
                    console.log('用户取消了生成');
                } else {
                    console.error('生成 AI 封面失败:', error);
                    showAiModal('AI 封面生成请求失败，请检查网络或稍后重试。');
                }
            } finally {
                // 无论成功失败还是取消，最后都重置状态
                // 注意：如果是取消操作，generateAbortController 已经在上面被置为 null 了
                // 如果是正常结束，这里需要置为 null
                if (generateAbortController) { 
                    generateAbortController = null;
                }
                updateGenerateButtonState();
            }
        });
    }

    updateAiCoverUI();
   
    // Function to update the state of all action buttons
    const updateButtonStates = () => {
        // If generating cover, disable save button
        if (generateAbortController) {
            saveBtn.textContent = '生成中...';
            saveBtn.disabled = true;
            saveBtn.style.opacity = '0.5';
            saveBtn.style.borderColor = '';
            saveBtn.style.color = '';
            openLink.classList.add('disabled');
            return;
        }

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

                if (data.cover_status === 'generating') {
                    isPersistedCoverGenerating = true;
                } else {
                    isPersistedCoverGenerating = false;
                }

                updateButtonStates(); // Re-check all button states
                updateAiCoverUI();
                
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
