import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamStatisticsPeriod } from './WalStreamStatisticsPeriod';

export const WAL_STREAM_STATISTICS_PERIOD_LABEL_KEYS: Record<
  WalStreamStatisticsPeriod,
  TranslationKey
> = {
  [WalStreamStatisticsPeriod.HOUR]: 'backups.physical.monitoring.periods.hour',
  [WalStreamStatisticsPeriod.DAY]: 'backups.physical.monitoring.periods.day',
  [WalStreamStatisticsPeriod.WEEK]: 'backups.physical.monitoring.periods.week',
};
