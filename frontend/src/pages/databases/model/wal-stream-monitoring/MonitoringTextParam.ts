import type { LocalizedText } from '../../../../shared/i18n';

export type MonitoringTextParam =
  | { kind: 'bytes'; bytes: number }
  | { kind: 'bytesPerSecond'; bytesPerSecond: number }
  | { kind: 'count'; count: number }
  | { kind: 'text'; text: LocalizedText }
  // A moment from today reads as a time of day, an older one with its date.
  | { kind: 'moment'; at: string; isToday: boolean };
