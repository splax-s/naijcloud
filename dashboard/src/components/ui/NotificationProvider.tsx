'use client';

import { useEffect, useState } from 'react';
import { 
  CheckCircleIcon, 
  ExclamationTriangleIcon, 
  InformationCircleIcon, 
  XCircleIcon,
  XMarkIcon 
} from '@heroicons/react/24/outline';
import { 
  Notification, 
  subscribe, 
  removeNotification, 
  getNotifications,
  NotificationType 
} from '@/lib/notifications';

function getIconForType(type: NotificationType) {
  switch (type) {
    case 'success':
      return CheckCircleIcon;
    case 'error':
      return XCircleIcon;
    case 'warning':
      return ExclamationTriangleIcon;
    case 'info':
    default:
      return InformationCircleIcon;
  }
}

function getColorClassesForType(type: NotificationType) {
  switch (type) {
    case 'success':
      return {
        container: 'bg-green-50 border-green-200',
        icon: 'text-green-400',
        title: 'text-green-800',
        message: 'text-green-700',
        close: 'text-green-500 hover:text-green-600',
      };
    case 'error':
      return {
        container: 'bg-red-50 border-red-200',
        icon: 'text-red-400',
        title: 'text-red-800',
        message: 'text-red-700',
        close: 'text-red-500 hover:text-red-600',
      };
    case 'warning':
      return {
        container: 'bg-yellow-50 border-yellow-200',
        icon: 'text-yellow-400',
        title: 'text-yellow-800',
        message: 'text-yellow-700',
        close: 'text-yellow-500 hover:text-yellow-600',
      };
    case 'info':
    default:
      return {
        container: 'bg-blue-50 border-blue-200',
        icon: 'text-blue-400',
        title: 'text-blue-800',
        message: 'text-blue-700',
        close: 'text-blue-500 hover:text-blue-600',
      };
  }
}

function NotificationItem({ notification }: { notification: Notification }) {
  const Icon = getIconForType(notification.type);
  const colors = getColorClassesForType(notification.type);

  return (
    <div className={`rounded-md border p-4 ${colors.container}`}>
      <div className="flex">
        <div className="flex-shrink-0">
          <Icon className={`h-5 w-5 ${colors.icon}`} aria-hidden="true" />
        </div>
        <div className="ml-3 flex-1">
          <h3 className={`text-sm font-medium ${colors.title}`}>
            {notification.title}
          </h3>
          <div className={`mt-1 text-sm ${colors.message}`}>
            <p>{notification.message}</p>
          </div>
        </div>
        <div className="ml-auto pl-3">
          <div className="-mx-1.5 -my-1.5">
            <button
              type="button"
              className={`inline-flex rounded-md p-1.5 focus:outline-none focus:ring-2 focus:ring-offset-2 ${colors.close}`}
              onClick={() => removeNotification(notification.id)}
            >
              <span className="sr-only">Dismiss</span>
              <XMarkIcon className="h-5 w-5" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export function NotificationProvider() {
  const [notifications, setNotifications] = useState<Notification[]>(getNotifications());

  useEffect(() => {
    const unsubscribe = subscribe(setNotifications);
    return unsubscribe;
  }, []);

  if (notifications.length === 0) {
    return null;
  }

  return (
    <div className="fixed top-0 right-0 z-50 p-4 space-y-4 w-full max-w-sm">
      {notifications.map((notification) => (
        <NotificationItem key={notification.id} notification={notification} />
      ))}
    </div>
  );
}
