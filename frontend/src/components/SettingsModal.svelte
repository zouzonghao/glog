<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { showNotification } from '../utils/notifications';

  export let title: string;
  export let settings: Record<string, any>;
  export let formId: string;
  export let fields: { name: string; label: string; type: string; placeholder?: string; min?: number }[];
  export let testEndpoint: string | null = null;
  export let backupEndpoint: string | null = null;
  
  let isOpen = false;
  let isLoading = false;
  let isTesting = false;
  let isBackingUp = false;
  let localSettings = { ...settings };

  const dispatch = createEventDispatcher();
  const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

  function openModal() {
    isOpen = true;
    dispatch('open');
  }

  function closeModal() {
    isOpen = false;
  }

  async function handleSave() {
    isLoading = true;
    const dataToSave: Record<string, any> = {};
    fields.forEach(field => {
      dataToSave[field.name] = localSettings[field.name];
    });

    // Filter out empty sensitive fields
    ['password', 'openai_token', 'imageapi_token', 'github_token', 'webdav_password'].forEach(key => {
      if (dataToSave[key] === '') {
        delete dataToSave[key];
      }
    });

    try {
      const response = await fetch(`${apiBaseUrl}/api/settings/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(dataToSave),
      });
      const result = await response.json();
      showNotification({ message: result.message || 'Error', type: response.ok ? 'success' : 'error' });
      if (response.ok) {
        closeModal();
      }
    } catch (error) {
      showNotification({ message: (error as Error).message, type: 'error' });
    } finally {
      isLoading = false;
    }
  }

  async function handleTestConnection() {
    if (!testEndpoint) return;
    isTesting = true;
    showNotification({ message: '测试中...', type: 'info' });

    const formData = new URLSearchParams();
    fields.forEach(field => {
        const value = localSettings[field.name];
        if (value !== undefined && value !== null) {
            formData.append(field.name, value.toString());
        }
    });

    try {
      const response = await fetch(`${apiBaseUrl}${testEndpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        credentials: 'include',
        body: formData.toString(),
      });
      const result = await response.json();
      showNotification({ message: result.message || 'Error', type: response.ok ? 'success' : 'error' });
    } catch (error) {
      showNotification({ message: (error as Error).message, type: 'error' });
    } finally {
      isTesting = false;
    }
  }

  async function handleBackupNow() {
    if (!backupEndpoint) return;
    isBackingUp = true;
    showNotification({ message: '正在备份...', type: 'info' });
    try {
        const response = await fetch(`${apiBaseUrl}${backupEndpoint}`, {
            method: 'POST',
            credentials: 'include',
        });
        const result = await response.json();
        showNotification({ message: result.message || 'Error', type: result.status || (response.ok ? 'success' : 'error') });
    } catch (error) {
        showNotification({ message: (error as Error).message, type: 'error' });
    } finally {
        isBackingUp = false;
    }
  }
</script>

<button type="button" class="btn" on:click={openModal}>
  <slot />
</button>

{#if isOpen}
<div
  class="modal-container show"
  role="button"
  tabindex="0"
  on:click|self={closeModal}
  on:keydown|self={(e) => { if (e.key === 'Enter' || e.key === ' ') closeModal() }}
>
  <div class="modal-content">
    <div class="modal-header">
      <h3>{title}</h3>
      <button on:click={closeModal} class="modal-close-btn">&times;</button>
    </div>
    <div class="modal-body">
      <form id={formId} on:submit|preventDefault={handleSave} class="app-form" autocomplete="off">
        {#each fields as field}
          <div class="settings-form-group">
            <label for={field.name}>{field.label}</label>
            <input 
              type={field.type} 
              id={field.name} 
              name={field.name}
              bind:value={localSettings[field.name]}
              placeholder={field.placeholder || ''}
              min={field.min}
              autocomplete={field.type === 'password' ? 'new-password' : 'off'}
              disabled={isLoading}
            >
          </div>
        {/each}
      </form>
    </div>
    <div class="modal-actions">
      <slot name="actions" {formId} />
      {#if backupEndpoint}
        <button type="button" on:click={handleBackupNow} class="btn" disabled={isBackingUp || isLoading}>
            {#if isBackingUp}⚡ 备份中...{:else}⚡ 立即备份{/if}
        </button>
      {/if}
      {#if testEndpoint}
        <button type="button" on:click={handleTestConnection} class="btn" disabled={isTesting || isLoading}>
            {#if isTesting}🚀 测试中...{:else}🚀 测试连接{/if}
        </button>
      {/if}
      <button type="button" on:click={handleSave} class="btn" disabled={isLoading || isTesting || isBackingUp}>
        {#if isLoading}
          保存中...
        {:else}
          💾 保存设置
        {/if}
      </button>
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
    border-bottom: 1px solid var(--border-color);
    padding-bottom: 0px;
    margin-bottom: 0px;
  }
  .modal-close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: var(--text-color);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    margin-top: 20px;
  }
</style>