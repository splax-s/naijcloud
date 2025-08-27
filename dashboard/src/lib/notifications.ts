// Simple notification system for user feedback
export type NotificationType = 'success' | 'error' | 'warning' | 'info';

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message: string;
  duration?: number;
}

// Simple in-memory notification store
let notifications: Notification[] = [];
let listeners: Array<(notifications: Notification[]) => void> = [];

export function addNotification(notification: Omit<Notification, 'id'>): string {
  const id = Math.random().toString(36).substr(2, 9);
  const newNotification: Notification = {
    ...notification,
    id,
    duration: notification.duration || 5000,
  };
  
  notifications = [...notifications, newNotification];
  notifyListeners();
  
  // Auto-remove after duration
  if (newNotification.duration && newNotification.duration > 0) {
    setTimeout(() => {
      removeNotification(id);
    }, newNotification.duration);
  }
  
  return id;
}

export function removeNotification(id: string): void {
  notifications = notifications.filter(n => n.id !== id);
  notifyListeners();
}

export function subscribe(listener: (notifications: Notification[]) => void): () => void {
  listeners.push(listener);
  return () => {
    listeners = listeners.filter(l => l !== listener);
  };
}

export function getNotifications(): Notification[] {
  return notifications;
}

function notifyListeners(): void {
  listeners.forEach(listener => listener(notifications));
}

// Helper functions for common notification types
export const notify = {
  success: (title: string, message: string) => 
    addNotification({ type: 'success', title, message }),
  
  error: (title: string, message: string) => 
    addNotification({ type: 'error', title, message }),
  
  warning: (title: string, message: string) => 
    addNotification({ type: 'warning', title, message }),
  
  info: (title: string, message: string) => 
    addNotification({ type: 'info', title, message }),
};
