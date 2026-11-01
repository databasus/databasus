import type { TranslationKey } from '../../../../shared/i18n';
import type { MonitoringTextParam } from './MonitoringTextParam';

// A sentence decided outside a component. Its amounts stay numbers and its durations stay keys, so
// the component formats and translates them in the current language while rendering.
export interface MonitoringText {
  key: TranslationKey;
  params?: Record<string, MonitoringTextParam>;
}
