import type { MonitoringText } from '@/pages/databases/model/wal-stream-monitoring/MonitoringText';
import type { WalStreamReasonLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonLevel';
import type { TranslationKey } from '@/shared/i18n';

export interface ReasonTexts {
  level: WalStreamReasonLevel;
  titleKey: TranslationKey;
  whatHappened: MonitoringText;
  whatToDo: MonitoringText;
}
