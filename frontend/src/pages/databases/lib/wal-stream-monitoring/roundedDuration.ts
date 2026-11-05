import type { LocalizedText } from '../../../../shared/i18n';

const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;
const SECONDS_PER_DAY = 86400;

// Minutes under an hour, hours and minutes under a day, days and hours beyond: precise enough to
// act on, short enough to read at a glance. A duration is never shown as zero minutes.
export const getRoundedDuration = (seconds: number): LocalizedText => {
  const totalMinutes = Math.max(1, Math.round(seconds / SECONDS_PER_MINUTE));

  if (totalMinutes < 60) {
    return {
      key: 'backups.physical.monitoring.durations.minutes',
      params: { minutes: totalMinutes },
    };
  }

  if (seconds < SECONDS_PER_DAY) {
    const hours = Math.floor(totalMinutes / 60);
    const minutes = totalMinutes % 60;

    if (hours < 24) {
      return {
        key: 'backups.physical.monitoring.durations.hoursMinutes',
        params: { hours, minutes },
      };
    }
  }

  const totalHours = Math.round(seconds / SECONDS_PER_HOUR);

  return {
    key: 'backups.physical.monitoring.durations.daysHours',
    params: { days: Math.floor(totalHours / 24), hours: totalHours % 24 },
  };
};

// The live label counts seconds, because "1 min" would hide whether the data is 5 or 55 seconds
// old.
export const getMeasurementAge = (seconds: number): LocalizedText => {
  if (seconds < SECONDS_PER_MINUTE) {
    return {
      key: 'backups.physical.monitoring.durations.seconds',
      params: { seconds: Math.max(0, Math.floor(seconds)) },
    };
  }

  return getRoundedDuration(seconds);
};

// Null when either time cannot be read, so a caller leaves the duration out instead of showing
// "NaN d NaN h".
export const getSecondsBetween = (earlier: string | Date, later: Date): number | null => {
  const earlierMs = new Date(earlier).getTime();
  const laterMs = later.getTime();

  if (!Number.isFinite(earlierMs) || !Number.isFinite(laterMs)) {
    return null;
  }

  return Math.max(0, (laterMs - earlierMs) / 1000);
};
