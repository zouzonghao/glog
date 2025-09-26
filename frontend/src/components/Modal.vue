<template>
  <transition name="modal-fade">
    <div v-if="show" class="modal-container" @click.self="close">
      <div class="modal-content">
        <span class="modal-close-btn" @click="close">&times;</span>
        <slot></slot>
      </div>
    </div>
  </transition>
</template>

<script setup>
const props = defineProps({
  show: {
    type: Boolean,
    required: true,
  },
});

const emit = defineEmits(['close']);

const close = () => {
  emit('close');
};
</script>

<style scoped>
.modal-container {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.6);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background-color: var(--bg-color);
  padding: 20px 30px;
  border-radius: 8px;
  position: relative;
  min-width: 400px;
  max-width: 90%;
  box-shadow: 0 5px 15px rgba(0,0,0,0.3);
}

.modal-close-btn {
  position: absolute;
  top: 10px;
  right: 15px;
  font-size: 24px;
  font-weight: bold;
  color: var(--text-muted);
  cursor: pointer;
  transition: color 0.2s;
}

.modal-close-btn:hover {
  color: var(--text-color);
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.3s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
</style>