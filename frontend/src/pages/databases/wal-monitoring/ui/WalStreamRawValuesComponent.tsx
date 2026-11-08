import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import { type WalStreamStatistics } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import {
  WalStreamSourceState,
  WAL_STREAM_SOURCE_STATE_LABEL_KEYS,
} from '@/pages/databases/wal-monitoring/model/WalStreamSourceState';
import { WAL_STREAM_STREAMER_STATE_LABEL_KEYS } from '@/pages/databases/wal-monitoring/model/WalStreamStreamerState';
import { type TranslationKey, useLocale } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib';
import { getUserTimeFormat } from '@/shared/time';

interface Props {
  statistics: WalStreamStatistics;
}

export const WalStreamRawValuesComponent = ({ statistics }: Props) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  const { current, streamer } = statistics;
  const notAvailable = t('backups.physical.monitoring.details.notAvailable');

  const formatMilliseconds = (milliseconds: number | null | undefined) =>
    milliseconds == null
      ? notAvailable
      : t('backups.physical.monitoring.details.milliseconds', {
          value: formatNumber(milliseconds),
        });

  const getAllowance = () => {
    if (current?.sourceState !== WalStreamSourceState.REACHABLE) {
      return notAvailable;
    }

    return current.slotSafeWalSizeBytes === null
      ? t('backups.physical.monitoring.details.noLimit')
      : formatBytes(current.slotSafeWalSizeBytes, formatNumber);
  };

  const getWalToStoredRatio = () =>
    current && current.archivedStoredBytes > 0
      ? formatNumber(current.archivedWalBytes / current.archivedStoredBytes, {
          maximumFractionDigits: 1,
        })
      : notAvailable;

  const rows: { labelKey: TranslationKey; value: string }[] = [
    {
      labelKey: 'backups.physical.monitoring.details.streamerState',
      value: t(WAL_STREAM_STREAMER_STATE_LABEL_KEYS[streamer.state]),
    },
    {
      labelKey: 'backups.physical.monitoring.details.sourceState',
      value: current ? t(WAL_STREAM_SOURCE_STATE_LABEL_KEYS[current.sourceState]) : notAvailable,
    },
    {
      labelKey: 'backups.physical.monitoring.details.slotStatus',
      value: current?.slotWalStatus ?? notAvailable,
    },
    { labelKey: 'backups.physical.monitoring.details.allowance', value: getAllowance() },
    {
      labelKey: 'backups.physical.monitoring.details.averageSegmentWrite',
      value: formatMilliseconds(current?.segmentWriteAvgMillis),
    },
    {
      labelKey: 'backups.physical.monitoring.details.longestSegmentWrite',
      value: formatMilliseconds(current?.segmentWriteMaxMillis),
    },
    {
      labelKey: 'backups.physical.monitoring.details.uploadConcurrency',
      value: current ? formatNumber(current.highestConcurrentUploads) : notAvailable,
    },
    {
      labelKey: 'backups.physical.monitoring.details.walToStoredRatio',
      value: getWalToStoredRatio(),
    },
    {
      labelKey: 'backups.physical.monitoring.details.walSegmentSize',
      value: current ? formatBytes(current.walSegmentSizeBytes, formatNumber) : notAvailable,
    },
    {
      labelKey: 'backups.physical.monitoring.details.sessionStart',
      value: streamer.sessionStartedAt
        ? dayjs(streamer.sessionStartedAt).format(getUserTimeFormat().format)
        : notAvailable,
    },
    {
      labelKey: 'backups.physical.monitoring.details.restartsLastHour',
      value: formatNumber(streamer.restartsLastHour),
    },
  ];

  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-2 text-sm md:grid-cols-2">
      {rows.map((row) => (
        <div key={row.labelKey} className="flex justify-between gap-4">
          <dt className="text-muted">{t(row.labelKey)}</dt>
          <dd className="text-right font-medium">{row.value}</dd>
        </div>
      ))}
    </dl>
  );
};
