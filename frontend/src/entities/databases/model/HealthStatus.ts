import type { ApiComponents } from '@/shared/api';

export type HealthStatus = ApiComponents['schemas']['DatabaseHealthStatus'];

export const HealthStatus = {
  AVAILABLE: 'AVAILABLE',
  UNAVAILABLE: 'UNAVAILABLE',
} as const satisfies Record<string, HealthStatus>;
