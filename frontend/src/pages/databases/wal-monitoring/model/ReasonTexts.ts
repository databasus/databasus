import type { MonitoringText } from '@/pages/databases/wal-monitoring/model/MonitoringText';
import type { WalStreamReasonLevel } from '@/pages/databases/wal-monitoring/model/WalStreamReasonLevel';
import type { TranslationKey } from '@/shared/i18n';

export interface ReasonTexts {
  level: WalStreamReasonLevel;
  titleKey: TranslationKey;
  whatHappened: MonitoringText;
  whatToDo: MonitoringText;
}
