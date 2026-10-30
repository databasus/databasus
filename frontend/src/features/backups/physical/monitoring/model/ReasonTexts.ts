import type { WalStreamReasonLevel } from '../../../../../entities/backups/physical';
import type { TranslationKey } from '../../../../../shared/i18n';
import type { MonitoringText } from './MonitoringText';

export interface ReasonTexts {
  level: WalStreamReasonLevel;
  titleKey: TranslationKey;
  whatHappened: MonitoringText;
  whatToDo: MonitoringText;
}
