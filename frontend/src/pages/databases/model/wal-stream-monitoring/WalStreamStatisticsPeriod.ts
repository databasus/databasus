import type { ApiComponents } from '../../../../shared/api';

export type WalStreamStatisticsPeriod = ApiComponents['schemas']['WalStreamStatisticsPeriod'];

export const WalStreamStatisticsPeriod = {
  HOUR: 'HOUR',
  DAY: 'DAY',
  WEEK: 'WEEK',
} as const satisfies Record<string, WalStreamStatisticsPeriod>;
