import type { ApiComponents } from '../../../shared/api';

export type ScheduleType = ApiComponents['schemas']['ScheduleType'];

export const ScheduleType = {
  HOURLY: 'HOURLY',
  DAILY: 'DAILY',
  WEEKLY: 'WEEKLY',
  MONTHLY: 'MONTHLY',
  CRON: 'CRON',
} as const satisfies Record<string, ScheduleType>;
