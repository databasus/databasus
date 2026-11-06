import dayjs from 'dayjs';
import type { TFunction } from 'i18next';

import type { MonitoringText } from '@/pages/databases/model/wal-stream-monitoring/MonitoringText';
import type { MonitoringTextParam } from '@/pages/databases/model/wal-stream-monitoring/MonitoringTextParam';
import { type LocaleContextValue, translateLocalizedText } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib';
import { getUserClockFormat, getUserTimeFormat } from '@/shared/time';

// Runs while a component renders, with that render's language, and turns the numbers and nested
// keys of a sentence into the values its translation interpolates.
export const getMonitoringTextValues = (
  text: MonitoringText,
  t: TFunction,
  formatNumber: LocaleContextValue['formatNumber'],
): Record<string, string> => {
  const values: Record<string, string> = {};

  for (const [name, param] of Object.entries(text.params ?? {})) {
    values[name] = formatMonitoringTextParam(param, t, formatNumber);
  }

  return values;
};

export const formatBytesPerSecond = (
  bytesPerSecond: number,
  t: TFunction,
  formatNumber: LocaleContextValue['formatNumber'],
): string =>
  t('backups.physical.monitoring.charts.perSecond', {
    amount: formatBytes(bytesPerSecond, formatNumber),
  });

const formatMonitoringTextParam = (
  param: MonitoringTextParam,
  t: TFunction,
  formatNumber: LocaleContextValue['formatNumber'],
): string => {
  switch (param.kind) {
    case 'bytes':
      return formatBytes(param.bytes, formatNumber);
    case 'bytesPerSecond':
      return formatBytesPerSecond(param.bytesPerSecond, t, formatNumber);
    case 'count':
      return formatNumber(param.count);
    case 'text':
      return translateLocalizedText(param.text, t);
    case 'moment':
      return formatMoment(param);
  }
};

export const formatMoment = ({ at, isToday }: { at: string; isToday: boolean }): string =>
  dayjs(at).format(isToday ? getUserClockFormat().format : getUserTimeFormat().format);
