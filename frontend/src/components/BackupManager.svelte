<script lang="ts">
  import { showNotification } from '../utils/notifications';
  import { showInputPrompt } from '../utils/prompts';
  import { onMount } from 'svelte';

  const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';
  const downloadUrl = `${apiBaseUrl}/api/backup/`;
  const uploadUrl = `${apiBaseUrl}/api/backup/upload`;

  let isUploading = false;
  let uploadInput: HTMLInputElement;
  let astroNavigate: ((href: string) => void) | null = null;

  onMount(() => {
    if ((window as any).Astro && typeof (window as any).Astro.navigate === 'function') {
      astroNavigate = (window as any).Astro.navigate;
    }
  });

  function handleNavigate(href: string) {
    if (astroNavigate) {
      astroNavigate(href);
    } else {
      console.warn("Astro.navigate not found, falling back to window.location.");
      window.location.href = href;
    }
  }

  function triggerUpload() {
    uploadInput.click();
  }

  async function handleFileUpload(event: Event) {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (!file) return;

    isUploading = true;
    const formData = new FormData();
    formData.append('backup', file);

    if (file.name.endsWith('.zip')) {
      const password = await showInputPrompt({ title: '请输入备份文件密码', inputType: 'password' });
      if (password === null) { // User cancelled the prompt
        isUploading = false;
        target.value = ''; // Reset file input
        return;
      }
      formData.append('password', password);
    }

    showNotification({ message: '正在上传并恢复...', type: 'info' });

    try {
      const response = await fetch(uploadUrl, {
        method: 'POST',
        credentials: 'include',
        body: formData,
      });
      const result = await response.json();
      if (response.ok) {
        sessionStorage.setItem('glog_notification', JSON.stringify({ message: result.message, type: result.status }));
        handleNavigate(window.location.href);
      } else {
        showNotification({ message: result.message, type: result.status });
      }
    } catch (error) {
      showNotification({ message: `上传失败: ${(error as Error).message}`, type: 'error' });
    } finally {
      isUploading = false;
      target.value = ''; // Reset file input
    }
  }
</script>

<span style="display: contents;">
  <a href={downloadUrl} class="btn">💾 下载备份</a>
  
  <input
    type="file"
    bind:this={uploadInput}
    on:change={handleFileUpload}
    accept=".zip,.json"
    class="hidden-file-input"
    disabled={isUploading}
  >
  <button
    type="button"
    on:click={triggerUpload}
    class="btn"
    disabled={isUploading}
  >
    {#if isUploading}
      上传中...
    {:else}
      📤 上传恢复
    {/if}
  </button>
</span>

<style>
  .hidden-file-input {
    display: none;
  }
</style>