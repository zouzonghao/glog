import { reactive } from 'vue';

export const notifications = reactive([]);

let nextId = 0;

export function showNotification(message, type = 'info', duration = 5000) {
  const id = nextId++;
  notifications.push({ id, message, type });

  setTimeout(() => {
    const index = notifications.findIndex(n => n.id === id);
    if (index !== -1) {
      notifications.splice(index, 1);
    }
  }, duration);
}