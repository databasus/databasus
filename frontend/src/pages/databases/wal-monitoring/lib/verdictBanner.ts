import dayjs from 'dayjs';

import {
  getReasonTexts,
  sortReasonsBrokenFirst,
} from '@/pages/databases/wal-monitoring/lib/reasonTexts';
import {
  getMeasurementAge,
  getRoundedDuration,
  getSecondsBetween,
} from '@/pages/databases/wal-monitoring/lib/roundedDuration';
import type {
  MonitoringText,
  MonitoringTextParam,
} from '@/pages/databases/wal-monitoring/model/MonitoringText';
import { MonitoringTone } from '@/pages/databases/wal-monitoring/model/MonitoringTone';
import {
  type VerdictBanner,
  VerdictIcon,
} from '@/pages/databases/wal-monitoring/model/VerdictBanner';
import { type WalStreamStatistics } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import {
  WalStreamHealthLevel,
  WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS,
} from '@/pages/databases/wal-monitoring/model/WalStreamHealthLevel';
import { WalStreamReasonCode } from '@/pages/databases/wal-monitoring/model/WalStreamReasonCode';

// Four missed refreshes: past this the screen no longer knows the current state, and showing the
// last verdict in its own color would claim otherwise.
export const UNCONFIRMED_STATE_AFTER_MS = 2 * 60 * 1000;

interface VerdictBannerInput {
  statistics: WalStreamStatistics;
  lastSuccessfulRefreshAt: Date;
  now: Date;
}

const LEVEL_TONES: Record<WalStreamHealthLevel, MonitoringTone> = {
  [WalStreamHealthLevel.HEALTHY]: MonitoringTone.SUCCESS,
  [WalStreamHealthLevel.NEEDS_ATTENTION]: MonitoringTone.WARNING,
  [WalStreamHealthLevel.BROKEN]: MonitoringTone.ERROR,
  [WalStreamHealthLevel.COLLECTING_DATA]: MonitoringTone.NEUTRAL,
  [WalStreamHealthLevel.NOT_STREAMING]: MonitoringTone.NEUTRAL,
};

const LEVEL_ICONS: Record<WalStreamHealthLevel, VerdictIcon> = {
  [WalStreamHealthLevel.HEALTHY]: VerdictIcon.CHECK,
  [WalStreamHealthLevel.NEEDS_ATTENTION]: VerdictIcon.EXCLAMATION,
  [WalStreamHealthLevel.BROKEN]: VerdictIcon.CLOSE,
  [WalStreamHealthLevel.COLLECTING_DATA]: VerdictIcon.CLOCK,
  [WalStreamHealthLevel.NOT_STREAMING]: VerdictIcon.PAUSE,
};

export const getVerdictBanner = ({
  statistics,
  lastSuccessfulRefreshAt,
  now,
}: VerdictBannerInput): VerdictBanner => {
  const { level } = statistics.health;
  const restoreSentence = getRestoreSentence(statistics, now);

  if (now.getTime() - lastSuccessfulRefreshAt.getTime() > UNCONFIRMED_STATE_AFTER_MS) {
    return {
      tone: MonitoringTone.NEUTRAL,
      icon: VerdictIcon.QUESTION,
      headlineKey: 'backups.physical.monitoring.banner.cannotConfirm',
      description: {
        key: 'backups.physical.monitoring.banner.cannotConfirmDescription',
        params: {
          level: { kind: 'text', text: { key: WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS[level] } },
          duration: {
            kind: 'text',
            text: getRoundedDuration(getSecondsBetween(lastSuccessfulRefreshAt, now) ?? 0),
          },
        },
      },
      otherReasons: [],
      restoreSentence,
      isConfigLinkShown: false,
      isStateConfirmed: false,
    };
  }

  const banner: VerdictBanner = {
    tone: LEVEL_TONES[level],
    icon: LEVEL_ICONS[level],
    headlineKey: WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS[level],
    otherReasons: [],
    restoreSentence,
    isConfigLinkShown: level === WalStreamHealthLevel.NOT_STREAMING,
    isStateConfirmed: true,
  };

  switch (level) {
    case WalStreamHealthLevel.HEALTHY:
      return { ...banner, headlineKey: 'backups.physical.monitoring.banner.healthyHeadline' };

    case WalStreamHealthLevel.COLLECTING_DATA:
      return {
        ...banner,
        description: { key: 'backups.physical.monitoring.banner.collectingDataDescription' },
      };

    case WalStreamHealthLevel.NOT_STREAMING:
      return {
        ...banner,
        description: { key: 'backups.physical.monitoring.banner.notStreamingDescription' },
      };

    default:
      return withReasons(banner, statistics, now);
  }
};

const withReasons = (
  banner: VerdictBanner,
  statistics: WalStreamStatistics,
  now: Date,
): VerdictBanner => {
  const [headlineReason, ...otherReasons] = sortReasonsBrokenFirst(statistics.health.reasons).map(
    (reason) => getReasonTexts(reason, statistics.health.isFullBackupPending, now),
  );

  if (!headlineReason) {
    return banner;
  }

  return { ...banner, headlineKey: headlineReason.titleKey, headlineReason, otherReasons };
};

const getRestoreSentence = (statistics: WalStreamStatistics, now: Date): MonitoringText => {
  const { latestRestorableAt, health } = statistics;

  if (latestRestorableAt === null) {
    return {
      key: health.isFullBackupPending
        ? 'backups.physical.monitoring.banner.restoreNothingPending'
        : 'backups.physical.monitoring.banner.restoreNothing',
    };
  }

  const moment = toMomentParam(latestRestorableAt, now);
  const ageSeconds = getSecondsBetween(latestRestorableAt, now);
  // The backend lists the broken chain reason only for a gap the latest restore cannot cross, so
  // the reason alone means saved changes lie beyond the restorable moment.
  const isLimitedByBrokenChain = health.reasons.some(
    (reason) => reason.code === WalStreamReasonCode.WAL_CHAIN_BROKEN,
  );
  // With a stretch that cannot be restored, "any moment up to" would promise the moments of that
  // stretch too, so the sentence names where the newest restorable stretch starts.
  const newestRange = statistics.restorableRanges[statistics.restorableRanges.length - 1];
  const from =
    !isLimitedByBrokenChain && statistics.restorableRanges.length > 1 && newestRange
      ? toMomentParam(newestRange.from, now)
      : null;

  if (ageSeconds === null) {
    if (from) {
      return {
        key: 'backups.physical.monitoring.banner.restoreFromUpToWithoutAge',
        params: { from, moment },
      };
    }

    return {
      key: isLimitedByBrokenChain
        ? 'backups.physical.monitoring.banner.restoreOnlyUpToWithoutAge'
        : 'backups.physical.monitoring.banner.restoreUpToWithoutAge',
      params: { moment },
    };
  }

  const age: MonitoringTextParam = { kind: 'text', text: getMeasurementAge(ageSeconds) };

  if (from) {
    return {
      key: 'backups.physical.monitoring.banner.restoreFromUpTo',
      params: { from, moment, age },
    };
  }

  return {
    key: isLimitedByBrokenChain
      ? 'backups.physical.monitoring.banner.restoreOnlyUpTo'
      : 'backups.physical.monitoring.banner.restoreUpTo',
    params: { moment, age },
  };
};

const toMomentParam = (at: string, now: Date): MonitoringTextParam => ({
  kind: 'moment',
  at,
  isToday: dayjs(at).isSame(now, 'day'),
});
