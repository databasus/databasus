import { NotifierType } from './NotifierType';

export const getNotifierLogoFromType = (type: NotifierType) => {
  switch (type) {
    case NotifierType.EMAIL:
      return '/icons/notifiers/email.svg';
    case NotifierType.TELEGRAM:
      return '/icons/notifiers/telegram.svg';
    case NotifierType.WEBHOOK:
      return '/icons/notifiers/webhook.svg';
    case NotifierType.SLACK:
      return '/icons/notifiers/slack.svg';
    case NotifierType.DISCORD:
      return '/icons/notifiers/discord.svg';
    case NotifierType.TEAMS:
      return '/icons/notifiers/teams.svg';
    case NotifierType.MATTERMOST:
      return '/icons/notifiers/mattermost.svg';
    case NotifierType.SES:
      return '/icons/notifiers/ses.svg';
    case NotifierType.SNS:
      return '/icons/notifiers/sns.svg';
    default:
      return '';
  }
};
