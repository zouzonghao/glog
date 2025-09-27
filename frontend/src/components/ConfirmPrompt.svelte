<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';

  export let title = '请确认';
  export let onConfirm: () => void;
  export let onCancel: () => void;

  let isOpen = true;

  const dispatch = createEventDispatcher();

  function handleConfirm() {
    onConfirm();
    close();
  }

  function handleCancel() {
    onCancel();
    close();
  }

  function close() {
    isOpen = false;
    setTimeout(() => dispatch('destroy'), 300);
  }
</script>

{#if isOpen}
<div 
  class="modal-container show" 
  role="dialog"
  aria-modal="true"
  aria-labelledby="confirm-prompt-title"
  tabindex="0"
  on:click|self={handleCancel}
  on:keydown={(e) => {
    if (e.key === 'Escape') handleCancel();
    if (e.key === 'Enter') handleConfirm();
  }}
>
  <div class="modal-content">
    <div class="modal-body">
      <p id="confirm-prompt-title">{title}</p>
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
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 2000;
  }
  .modal-content {
    background-color: var(--color-background);
    padding: 20px;
    border-radius: 8px;
    width: 90%;
    max-width: 400px;
    box-shadow: 0 5px 15px rgba(0,0,0,0.3);
  }
  .modal-body p {
    margin: 0;
    font-size: 1.1rem;
    text-align: center;
    margin-top: 10px;
  }
  .modal-actions {
    display: flex;
    justify-content: center;
    gap: 10px;
    margin-top: 10px;
    margin-bottom: 5px;
  }
</style>