<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';

  export let title = '请输入';
  export let inputType = 'text';
  export let onConfirm: (value: string) => void;
  export let onCancel: () => void;

  let value = '';
  let inputElement: HTMLInputElement;
  let isOpen = true;

  const dispatch = createEventDispatcher();

  onMount(() => {
    inputElement?.focus();
  });

  function handleConfirm() {
    onConfirm(value);
    close();
  }

  function handleCancel() {
    onCancel();
    close();
  }

  function close() {
    isOpen = false;
    // Give time for animation before destroying
    setTimeout(() => dispatch('destroy'), 300);
  }
</script>

{#if isOpen}
<div 
  class="modal-container show" 
  role="dialog"
  aria-modal="true"
  aria-labelledby="input-prompt-title"
  tabindex="0"
  on:click|self={handleCancel}
  on:keydown={(e) => {
    if (e.key === 'Escape') handleCancel();
    if (e.key === 'Enter') handleConfirm();
  }}
>
  <div class="modal-content">
    <div class="modal-header">
      <h3 id="input-prompt-title">{title}</h3>
      <button on:click={handleCancel} class="modal-close-btn">&times;</button>
    </div>
    <div class="modal-body">
      <form on:submit|preventDefault={handleConfirm} class="app-form" autocomplete="off">
        <div class="settings-form-group">
          <input 
            type={inputType}
            bind:value={value}
            bind:this={inputElement}
            autocomplete="off"
          >
        </div>
      </form>
    </div>
    <div class="modal-actions">
      <button type="button" on:click={handleCancel} class="btn btn-secondary">取消</button>
      <button type="button" on:click={handleConfirm} class="btn">确认</button>
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
    z-index: 2000;
  }
  .modal-container.show {
    display: flex;
  }
  .modal-content {
    background-color: var(--color-background);
    padding: 20px;
    border-radius: 8px;
    width: 90%;
    max-width: 400px;
    box-shadow: 0 5px 15px rgba(0,0,0,0.3);
  }
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--color-border-primary);
    padding-bottom: 10px;
    margin-bottom: 20px;
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
  input {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid var(--color-border-primary);
    border-radius: 4px;
    background-color: var(--color-background-input);
    color: var(--color-text-primary);
    font-size: 0.9rem;
  }
  input:focus {
    outline: none;
    border-color: var(--color-accent-primary);
  }
</style>