export type NotificationType = 'info' | 'success' | 'error';

export interface NotificationDetail {
    message: string;
    type?: NotificationType;
    duration?: number;
}

export function showNotification(detail: NotificationDetail) {
    const event = new CustomEvent('show-notification', { detail });
    document.dispatchEvent(event);
}