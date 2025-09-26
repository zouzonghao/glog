<template>
  <div id="notification-container">
    <transition-group name="notification" tag="div">
      <div v-for="notification in notifications" :key="notification.id" :class="['notification', notification.type]">
        {{ notification.message }}
      </div>
    </transition-group>
  </div>
</template>

<script setup>
import { notifications } from '@/services/notification';
</script>

<style scoped>
#notification-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1050;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.notification {
  background-color: var(--bg-secondary);
  color: var(--text-color);
  padding: 15px 20px;
  border-radius: 5px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  border-left: 5px solid var(--primary-color);
  opacity: 0;
  transform: translateX(100%);
  transition: all 0.5s cubic-bezier(0.68, -0.55, 0.27, 1.55);
}

.notification.info {
  border-left-color: var(--primary-color);
}

.notification.success {
  border-left-color: #28a745;
}

.notification.error {
  border-left-color: #dc3545;
}

/* Enter and leave animations */
.notification-enter-active,
.notification-leave-active {
  transition: all 0.5s ease;
}
.notification-enter-from,
.notification-leave-to {
  opacity: 0;
  transform: translateX(30px);
}
</style>