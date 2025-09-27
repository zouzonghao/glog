<script lang="ts">
  import { login } from '../services/api';
  import { showNotification } from '../utils/notifications';
  import { onMount } from 'svelte';

  let password = '';
  let isLoading = false;
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

  async function handleSubmit() {
    if (!password) {
      showNotification({ message: '请输入密码', type: 'error' });
      return;
    }

    isLoading = true;

    try {
      await login(password);
      sessionStorage.setItem('glog_notification', JSON.stringify({ message: '登录成功!', type: 'success' }));
      handleNavigate('/admin');
    } catch (error) {
      if (error instanceof Error) {
        showNotification({ message: error.message, type: 'error' });
      } else {
        showNotification({ message: '发生未知错误', type: 'error' });
      }
    } finally {
      isLoading = false;
    }
  }
</script>

<form on:submit|preventDefault={handleSubmit} class="editor-form app-form">
  <div class="form-group login-form-group">
    <input 
      type="password" 
      bind:value={password}
      required 
      autocomplete="current-password" 
      placeholder="请输入密码" 
      class="login-password-input"
      disabled={isLoading}
    >
    <button 
      type="submit" 
      class="login-submit-btn" 
      disabled={isLoading}
    >
      {#if isLoading}
        登录中...
      {:else}
        登录
      {/if}
    </button>
  </div>
</form>

<style>
  .login-form-group {
    display: flex;
  }
  .login-password-input {
    flex-grow: 1;
    border-right: none;
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
  }
  .login-submit-btn {
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
  }
</style>