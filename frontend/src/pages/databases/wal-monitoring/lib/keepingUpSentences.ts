import { getRoundedDuration } from '@/pages/databases/wal-monitoring/lib/roundedDuration';
import type { MonitoringText } from '@/pages/databases/wal-monitoring/model/MonitoringText';
import { type WalStreamKeepingUp } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import { WalStreamKeepingUpState } from '@/pages/databases/wal-monitoring/model/WalStreamKeepingUpState';

const SECONDS_PER_HOUR = 3600;

// A consequence further away than this is not news yet, and saying it would only alarm.
const CONSEQUENCE_HORIZON_SECONDS = 24 * SECONDS_PER_HOUR;

// One or two plain sentences on whether archiving keeps up, then what happens next when the
// source or Databasus will act within a day.
export const getKeepingUpSentences = (keepingUp: WalStreamKeepingUp): MonitoringText[] => [
  ...getStateSentences(keepingUp),
  ...getConsequenceSentences(keepingUp),
];

const getStateSentences = (keepingUp: WalStreamKeepingUp): MonitoringText[] => {
  const writeRate = keepingUp.sourceWriteBytesPerSecond ?? 0;
  const archiveRate = keepingUp.archiveBytesPerSecond ?? 0;

  switch (keepingUp.state) {
    case WalStreamKeepingUpState.KEEPING_UP:
      return [
        {
          key: 'backups.physical.monitoring.keepingUp.keepingUp',
          params: {
            writeRate: { kind: 'bytesPerSecond', bytesPerSecond: writeRate },
            archiveRate: { kind: 'bytesPerSecond', bytesPerSecond: archiveRate },
          },
        },
      ];

    case WalStreamKeepingUpState.FALLING_BEHIND:
      return getFallingBehindSentences(keepingUp, writeRate, archiveRate);

    case WalStreamKeepingUpState.CATCHING_UP:
      return [getCatchingUpSentence(keepingUp)];

    case WalStreamKeepingUpState.IDLE:
      return [{ key: 'backups.physical.monitoring.keepingUp.idle' }];

    // The figures above already say "not measured", so an unknown state adds no sentence.
    default:
      return [];
  }
};

const getFallingBehindSentences = (
  keepingUp: WalStreamKeepingUp,
  writeRate: number,
  archiveRate: number,
): MonitoringText[] => {
  const sentences: MonitoringText[] = [
    {
      key: 'backups.physical.monitoring.keepingUp.fallingBehind',
      params: {
        differenceRate: {
          kind: 'bytesPerSecond',
          bytesPerSecond: Math.max(0, writeRate - archiveRate),
        },
        writeRate: { kind: 'bytesPerSecond', bytesPerSecond: writeRate },
        archiveRate: { kind: 'bytesPerSecond', bytesPerSecond: archiveRate },
      },
    },
  ];

  if (
    keepingUp.unarchivedWalBytes !== null &&
    keepingUp.unarchivedWalChangeBytesPerSecond !== null &&
    keepingUp.unarchivedWalChangeBytesPerSecond > 0
  ) {
    sentences.push({
      key: 'backups.physical.monitoring.keepingUp.fallingBehindBacklog',
      params: {
        unsavedAmount: { kind: 'bytes', bytes: keepingUp.unarchivedWalBytes },
        growthPerHour: {
          kind: 'bytes',
          bytes: keepingUp.unarchivedWalChangeBytesPerSecond * SECONDS_PER_HOUR,
        },
      },
    });
  }

  return sentences;
};

const getCatchingUpSentence = (keepingUp: WalStreamKeepingUp): MonitoringText => {
  const unsavedAmount = { kind: 'bytes' as const, bytes: keepingUp.unarchivedWalBytes ?? 0 };

  if (keepingUp.estimatedCatchUpSeconds === null) {
    return {
      key: 'backups.physical.monitoring.keepingUp.catchingUpWithoutEstimate',
      params: { unsavedAmount },
    };
  }

  return {
    key: 'backups.physical.monitoring.keepingUp.catchingUp',
    params: {
      unsavedAmount,
      duration: { kind: 'text', text: getRoundedDuration(keepingUp.estimatedCatchUpSeconds) },
    },
  };
};

const getConsequenceSentences = (keepingUp: WalStreamKeepingUp): MonitoringText[] => {
  const sentences: MonitoringText[] = [];

  if (isWithinConsequenceHorizon(keepingUp.estimatedSlotRebuildSeconds)) {
    sentences.push({
      key: 'backups.physical.monitoring.keepingUp.rebuildConsequence',
      params: {
        duration: {
          kind: 'text',
          text: getRoundedDuration(keepingUp.estimatedSlotRebuildSeconds ?? 0),
        },
      },
    });
  }

  if (isWithinConsequenceHorizon(keepingUp.estimatedSlotLimitSeconds)) {
    sentences.push({
      key: 'backups.physical.monitoring.keepingUp.limitConsequence',
      params: {
        duration: {
          kind: 'text',
          text: getRoundedDuration(keepingUp.estimatedSlotLimitSeconds ?? 0),
        },
      },
    });
  }

  return sentences;
};

const isWithinConsequenceHorizon = (estimatedSeconds: number | null): boolean =>
  estimatedSeconds !== null && estimatedSeconds < CONSEQUENCE_HORIZON_SECONDS;
