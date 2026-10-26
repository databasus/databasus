import {
  type NotifierSettings,
  NotifierType,
  validateDiscordNotifier,
  validateEmailNotifier,
  validateMattermostNotifier,
  validateSlackNotifier,
  validateTeamsNotifier,
  validateTelegramNotifier,
  validateWebhookNotifier,
} from '../../../entity/notifiers';

// A saved notifier may leave its secrets empty, because the backend keeps the stored ones.
export const isNotifierSettingsFilled = (settings: NotifierSettings, isSaved: boolean): boolean => {
  const isCreate = !isSaved;

  switch (settings.type) {
    case NotifierType.EMAIL:
      return validateEmailNotifier(settings);
    case NotifierType.TELEGRAM:
      return validateTelegramNotifier(isCreate, settings);
    case NotifierType.SLACK:
      return validateSlackNotifier(isCreate, settings);
    case NotifierType.DISCORD:
      return validateDiscordNotifier(isCreate, settings);
    case NotifierType.TEAMS:
      return validateTeamsNotifier(isCreate, settings);
    case NotifierType.MATTERMOST:
      return validateMattermostNotifier(isCreate, settings);
    case NotifierType.WEBHOOK:
      return validateWebhookNotifier(settings);
  }
};
