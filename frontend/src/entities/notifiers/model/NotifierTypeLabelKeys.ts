import type { TranslationKey } from '../../../shared/i18n';
import { NotifierType } from './NotifierType';

export const NOTIFIER_TYPE_LABEL_KEYS: Record<NotifierType, TranslationKey> = {
  [NotifierType.EMAIL]: 'notifiers.types.email',
  [NotifierType.TELEGRAM]: 'notifiers.types.telegram',
  [NotifierType.WEBHOOK]: 'notifiers.types.webhook',
  [NotifierType.SLACK]: 'notifiers.types.slack',
  [NotifierType.DISCORD]: 'notifiers.types.discord',
  [NotifierType.TEAMS]: 'notifiers.types.teams',
  [NotifierType.MATTERMOST]: 'notifiers.types.mattermost',
};
