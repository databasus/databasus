import type { TranslationKey } from '../../../../shared/i18n';
import type { MonitoringText } from './MonitoringText';
import type { WalStreamReasonLevel } from './WalStreamReasonLevel';

export interface ReasonTexts {
  level: WalStreamReasonLevel;
  titleKey: TranslationKey;
  whatHappened: MonitoringText;
  whatToDo: MonitoringText;
}
