import type { MonitoringText } from '@/pages/databases/wal-monitoring/model/MonitoringText';
import type { MonitoringTone } from '@/pages/databases/wal-monitoring/model/MonitoringTone';
import type { ReasonTexts } from '@/pages/databases/wal-monitoring/model/ReasonTexts';
import type { TranslationKey } from '@/shared/i18n';

export interface VerdictBanner {
  tone: MonitoringTone;
  icon: VerdictIcon;
  headlineKey: TranslationKey;
  description?: MonitoringText;

  // The most severe reason names the headline; its sentences sit right under it, and the other
  // reasons follow as compact lines.
  headlineReason?: ReasonTexts;
  otherReasons: ReasonTexts[];

  restoreSentence: MonitoringText;
  isConfigLinkShown: boolean;

  // False once refreshes have failed for too long to vouch for the verdict on screen.
  isStateConfirmed: boolean;
}

export enum VerdictIcon {
  CHECK = 'CHECK',
  EXCLAMATION = 'EXCLAMATION',
  CLOSE = 'CLOSE',
  CLOCK = 'CLOCK',
  PAUSE = 'PAUSE',
  QUESTION = 'QUESTION',
}
