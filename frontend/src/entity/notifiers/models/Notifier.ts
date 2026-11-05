import type { DiscordNotifier } from './discord/DiscordNotifier';
import type { EmailNotifier } from './email/EmailNotifier';
import type { MattermostNotifier } from './mattermost/MattermostNotifier';
import type { NotifierType } from './NotifierType';
import type { SESNotifier } from './ses/SESNotifier';
import type { SlackNotifier } from './slack/SlackNotifier';
import type { SNSNotifier } from './sns/SNSNotifier';
import type { TeamsNotifier } from './teams/TeamsNotifier';
import type { TelegramNotifier } from './telegram/TelegramNotifier';
import type { WebhookNotifier } from './webhook/WebhookNotifier';

export interface Notifier {
  id: string;
  name: string;
  notifierType: NotifierType;
  lastSendError?: string;
  workspaceId: string;

  // specific notifier
  telegramNotifier?: TelegramNotifier;
  emailNotifier?: EmailNotifier;
  webhookNotifier?: WebhookNotifier;
  slackNotifier?: SlackNotifier;
  discordNotifier?: DiscordNotifier;
  teamsNotifier?: TeamsNotifier;
  mattermostNotifier?: MattermostNotifier;
  sesNotifier?: SESNotifier;
  snsNotifier?: SNSNotifier;
}
