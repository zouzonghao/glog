<script lang="ts">
  import { showNotification } from '../utils/notifications';
  import { onMount } from 'svelte';

  export let settings: Record<string, any>;

  let isLoading = false;
  let password = '';
  let astroNavigate: ((href: string) => void) | null = null;
  
  // Create a local copy for form binding
  let localSettings = { ...settings };

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

  async function saveSiteInfo() {
    isLoading = true;
    const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

    const dataToSave = {
      ...localSettings,
      password: password || undefined, // Only send password if it's not empty
    };

    // Remove undefined password field
    if (dataToSave.password === undefined) {
      delete dataToSave.password;
    }

    try {
      const response = await fetch(`${apiBaseUrl}/api/settings/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(dataToSave),
      });
      const result = await response.json();
      if (response.ok) {
        sessionStorage.setItem('glog_notification', JSON.stringify({ message: result.message || '设置已成功保存！', type: 'success' }));
        handleNavigate(window.location.href);
      } else {
        showNotification({ message: result.message || 'Error', type: 'error' });
      }
    } catch (error) {
      showNotification({ message: (error as Error).message, type: 'error' });
    } finally {
      isLoading = false;
    }
  }
</script>

<form on:submit|preventDefault={saveSiteInfo} class="app-form" autocomplete="off">
  <div class="settings-form-group settings-form-group-spaced">
    <label for="password">管理员密码</label>
    <input type="password" id="password" bind:value={password} placeholder="留空则不修改" autocomplete="new-password" disabled={isLoading}>
  </div>
  <div class="settings-form-group">
    <label for="favicon">浏览器标签页图标（url）</label>
    <input type="text" id="favicon" bind:value={localSettings.favicon} autocomplete="off" disabled={isLoading}>
  </div>
  <div class="settings-form-group">
    <label for="site_title">站点标题</label>
    <input type="text" id="site_title" bind:value={localSettings.site_title} autocomplete="off" disabled={isLoading}>
  </div>
  <div class="settings-form-group">
    <label for="site_description">站点描述</label>
    <input type="text" id="site_description" bind:value={localSettings.site_description} autocomplete="off" disabled={isLoading}>
  </div>
  <div class="settings-form-group">
    <label for="cover_prefix">封面前缀</label>
    <input type="text" id="cover_prefix" bind:value={localSettings.cover_prefix} autocomplete="off" disabled={isLoading}>
  </div>
  <div class="settings-actions settings-form-group-spaced">
    <button type="submit" class="btn" disabled={isLoading}>
      {#if isLoading}
        保存中...
      {:else}
        💾 保存站点信息
      {/if}
    </button>
  </div>
</form>