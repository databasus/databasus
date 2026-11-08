import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type WalStreamStatisticsPeriod = ApiComponents['schemas']['WalStreamStatisticsPeriod'];

export const WalStreamStatisticsPeriod = {
  HOUR: 'HOUR',
  DAY: 'DAY',
  WEEK: 'WEEK',
} as const satisfies Record<string, WalStreamStatisticsPeriod>;

export const WAL_STREAM_STATISTICS_PERIOD_LABEL_KEYS: Record<
  WalStreamStatisticsPeriod,
  TranslationKey
> = {
  [WalStreamStatisticsPeriod.HOUR]: 'backups.physical.monitoring.periods.hour',
  [WalStreamStatisticsPeriod.DAY]: 'backups.physical.monitoring.periods.day',
  [WalStreamStatisticsPeriod.WEEK]: 'backups.physical.monitoring.periods.week',
};
