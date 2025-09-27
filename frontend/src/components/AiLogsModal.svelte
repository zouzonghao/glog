<script lang="ts">
  import { showNotification } from '../utils/notifications';

  let isOpen = false;
  let isLoading = false;
  let logs = '正在加载日志...';
  const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

  async function fetchLogs() {
    isLoading = true;
    logs = '正在加载日志...';
    try {
      const response = await fetch(`${apiBaseUrl}/api/ai-logs/`, { credentials: 'include' });
      const result = await response.json();
      logs = result.status === 'success' ? (result.logs || '暂无 AI 日志。') : `加载日志失败: ${result.message}`;
    } catch (error) {
      logs = `加载日志时发生网络错误: ${(error as Error).message}`;
    } finally {
      isLoading = false;
    }
  }

  async function clearLogs() {
    showNotification({ message: '正在清除日志...', type: 'info' });
    try {
      const response = await fetch(`${apiBaseUrl}/api/ai-logs/clear`, {
        method: 'POST',
        credentials: 'include',
      });
      const result = await response.json();
      showNotification({ message: result.message, type: result.status });
      if (result.status === 'success') {
        logs = '日志已清除。';
      }
    } catch (error) {
      showNotification({ message: `清除日志时发生网络错误: ${(error as Error).message}`, type: 'error' });
    }
  }

  function openModal() {
    isOpen = true;
    fetchLogs();
  }

  function closeModal() {
    isOpen = false;
  }
</script>

<button type="button" class="btn" on:click={openModal}>
  📜 AI 日志
</button>

{#if isOpen}
<div 
  class="modal-container show" 
  role="dialog"
  aria-modal="true"
  aria-labelledby="ai-logs-title"
  tabindex="0"
  on:click|self={closeModal}
  on:keydown|self={(e) => { if (e.key === 'Escape') closeModal() }}
>
  <div class="modal-content" style="max-width: 700px;">
    <div class="modal-header">
      <h3 id="ai-logs-title">AI 任务日志</h3>
      <button on:click={closeModal} class="modal-close-btn">&times;</button>
    </div>
    <div class="modal-body">
      <div class="logs-container">
        <pre>{logs}</pre>
      </div>
    </div>
    <div class="modal-actions">
      <button type="button" on:click={clearLogs} class="btn">🗑️ 清除日志</button>
    </div>
  </div>
</div>
{/if}

<style>
  .modal-container {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.5);
    display: none;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }
  .modal-container.show {
    display: flex;
  }
  .modal-content {
    background-color: var(--color-background);
    padding: 20px;
    border-radius: 8px;
    width: 90%;
    max-width: 500px;
    box-shadow: 0 5px 15px rgba(0,0,0,0.3);
  }
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--color-border-primary);
    padding-bottom: 0px;
    margin-bottom: 0px;
  }
  .modal-close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: var(--color-text-primary);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    margin-top: 20px;
  }
  .logs-container {
    margin-top: 1rem;
    background-color: #1a1a1a;
    border: 1px solid #444;
    border-radius: 6px;
    padding: 1rem;
    height: 300px;
    overflow-y: auto;
    text-align: left;
  }
  pre {
    font-family: var(--font-mono);
    font-size: 0.85rem;
    color: #e0e0e0;
    white-space: pre-wrap;
    word-wrap: break-word;
    margin: 0;
  }
</style>