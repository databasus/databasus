import { WalStreamStatisticsPeriod } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatisticsPeriod';
import type { TranslationKey } from '@/shared/i18n';

export const WAL_STREAM_STATISTICS_PERIOD_LABEL_KEYS: Record<
  WalStreamStatisticsPeriod,
  TranslationKey
> = {
  [WalStreamStatisticsPeriod.HOUR]: 'backups.physical.monitoring.periods.hour',
  [WalStreamStatisticsPeriod.DAY]: 'backups.physical.monitoring.periods.day',
  [WalStreamStatisticsPeriod.WEEK]: 'backups.physical.monitoring.periods.week',
};
