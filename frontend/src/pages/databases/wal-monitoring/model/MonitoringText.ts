import type { TranslationKey, LocalizedText } from '@/shared/i18n';

// A sentence decided outside a component. Its amounts stay numbers and its durations stay keys, so
// the component formats and translates them in the current language while rendering.
export interface MonitoringText {
  key: TranslationKey;
  params?: Record<string, MonitoringTextParam>;
}

export type MonitoringTextParam =
  | { kind: 'bytes'; bytes: number }
  | { kind: 'bytesPerSecond'; bytesPerSecond: number }
  | { kind: 'count'; count: number }
  | { kind: 'text'; text: LocalizedText }
  // A moment from today reads as a time of day, an older one with its date.
  | { kind: 'moment'; at: string; isToday: boolean };
