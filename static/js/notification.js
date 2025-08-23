function showNotification(message, type = 'info') {
    const container = document.getElementById('notification-container');
    if (!container) {
        console.error('Notification container not found.');
        return;
    }

    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;

    container.appendChild(notification);

    // Animate in
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);

    // Automatically remove after 5 seconds
    setTimeout(() => {
        notification.classList.remove('show');
        // Remove the element after the transition ends
        notification.addEventListener('transitionend', () => {
            notification.remove();
        });
    }, 5000);
}