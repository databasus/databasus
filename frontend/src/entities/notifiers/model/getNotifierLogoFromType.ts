import { NotifierType } from '@/entities/notifiers/model/NotifierType';

const NOTIFIER_LOGOS: Record<NotifierType, string> = {
  [NotifierType.EMAIL]: '/icons/notifiers/email.svg',
  [NotifierType.TELEGRAM]: '/icons/notifiers/telegram.svg',
  [NotifierType.WEBHOOK]: '/icons/notifiers/webhook.svg',
  [NotifierType.SLACK]: '/icons/notifiers/slack.svg',
  [NotifierType.DISCORD]: '/icons/notifiers/discord.svg',
  [NotifierType.TEAMS]: '/icons/notifiers/teams.svg',
  [NotifierType.MATTERMOST]: '/icons/notifiers/mattermost.svg',
};

export const getNotifierLogoFromType = (type: NotifierType): string => NOTIFIER_LOGOS[type];
