import type { MonitoringTextParam } from '@/pages/databases/model/wal-stream-monitoring/MonitoringTextParam';
import type { TranslationKey } from '@/shared/i18n';

// A sentence decided outside a component. Its amounts stay numbers and its durations stay keys, so
// the component formats and translates them in the current language while rendering.
export interface MonitoringText {
  key: TranslationKey;
  params?: Record<string, MonitoringTextParam>;
}
