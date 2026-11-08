import type { ReactNode } from 'react';

export interface FeedbackNotice {
  kind: 'success' | 'error' | 'warning' | 'info';
  title: string;
  description?: ReactNode;
  lifetimeMs?: number;
  position?: 'top-center' | 'top-right' | 'bottom-right';
  isSessionTransition?: boolean;
}
