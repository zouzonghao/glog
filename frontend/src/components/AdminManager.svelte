<script lang="ts">
  import { onMount, tick } from 'svelte';
  import type { Post } from '../services/api';
  import { showConfirmPrompt } from '../utils/prompts';

  export let posts: Post[];

  let selectAll = false;
  let selectedIds: number[] = [];
  let astroNavigate: ((href: string) => void) | null = null;
  const apiBaseUrl = import.meta.env.PUBLIC_API_URL || '';

  onMount(() => {
    if ((window as any).Astro && typeof (window as any).Astro.navigate === 'function') {
      astroNavigate = (window as any).Astro.navigate;
    }
  });

  function handleNavigate(href: string) {
    if (astroNavigate) {
      astroNavigate(href);
    } else {
      window.location.href = href;
    }
  }

  function toggleSelectAll() {
    if (selectAll) {
      selectedIds = posts.map(p => p.id);
    } else {
      selectedIds = [];
    }
  }

  async function handleBatchAction(action: 'delete' | 'set-private', isPrivate?: boolean) {
    if (selectedIds.length === 0) {
      sessionStorage.setItem('glog_notification', JSON.stringify({ message: '请至少选择一篇文章', type: 'info' }));
      handleNavigate(window.location.href);
      return;
    }

    if (action === 'delete') {
      const confirmed = await showConfirmPrompt('确定要删除选中的文章吗？');
      if (!confirmed) return; // User cancelled
    }

    try {
      const payload = {
        ids: selectedIds,
        action: action,
        is_private: isPrivate,
      };

      console.log('Sending batch update payload:', payload); // Debugging log

      const response = await fetch(`${apiBaseUrl}/api/posts/batch-update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(payload),
      });
      const result = await response.json();
      sessionStorage.setItem('glog_notification', JSON.stringify({ message: result.message, type: result.status }));
      handleNavigate(window.location.href);
    } catch (error) {
      sessionStorage.setItem('glog_notification', JSON.stringify({ message: (error as Error).message, type: 'error' }));
      handleNavigate(window.location.href);
    }
  }

  async function handleSingleDelete(postId: number) {
    const confirmed = await showConfirmPrompt(`确定要删除这篇文章吗？`);
    if (!confirmed) return;

    try {
      const response = await fetch(`${apiBaseUrl}/api/posts/${postId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      const result = await response.json();
      sessionStorage.setItem('glog_notification', JSON.stringify({ message: result.message, type: result.status }));
      handleNavigate(window.location.href);
    } catch (error) {
      sessionStorage.setItem('glog_notification', JSON.stringify({ message: (error as Error).message, type: 'error' }));
      handleNavigate(window.location.href);
    }
  }

  $: {
    if (posts && posts.length > 0 && selectedIds.length === posts.length) {
      selectAll = true;
    } else {
      selectAll = false;
    }
  }
</script>

<div class="post-list-container">
  <div class="post-list-header">
    <div class="col-checkbox">
      <input type="checkbox" bind:checked={selectAll} on:change={toggleSelectAll} title="全选">
    </div>
    <div class="col-title">标题</div>
    <div class="col-private">私密</div>
    <div class="col-date">发布日期</div>
    <div class="col-actions">操作</div>
  </div>
  <div class="post-list-body">
    {#if posts.length > 0}
      {#each posts as post (post.id)}
        <div class="post-list-item">
          <div class="col-checkbox">
            <input type="checkbox" bind:group={selectedIds} value={post.id}>
          </div>
          <div class="col-title" title={post.title}>
            <a href={`/post/${post.slug}`}>{post.title}</a>
          </div>
          <div class="col-private">
            {post.is_private ? '是' : '-'}
          </div>
          <div class="col-date">
            {#if post.published_at}
              {new Date(post.published_at).toISOString().split('T')[0]}
            {:else}
              —
            {/if}
          </div>
          <div class="col-actions">
            <a href={`/admin/editor?id=${post.id}`}>[编辑]</a>
            <button class="link-button" on:click={() => handleSingleDelete(post.id)}>[删除]</button>
          </div>
        </div>
      {/each}
    {:else}
      <div class="empty-state">
        <p>没有文章。</p>
      </div>
    {/if}
  </div>
</div>

<div class="batch-actions-container">
  <button on:click={() => handleBatchAction('delete')} class="btn" disabled={selectedIds.length === 0}>批量删除</button>
  <button on:click={() => handleBatchAction('set-private', true)} class="btn" disabled={selectedIds.length === 0}>设为私密</button>
  <button on:click={() => handleBatchAction('set-private', false)} class="btn" disabled={selectedIds.length === 0}>设为公开</button>
</div>
<style>
  .link-button {
    background: none;
    border: none;
    color: var(--link-color);
    cursor: pointer;
    padding: 0;
    font-size: inherit;
    font-family: inherit;
    text-decoration: none;
  }
  .link-button:hover {
    text-decoration: underline;
  }
</style>