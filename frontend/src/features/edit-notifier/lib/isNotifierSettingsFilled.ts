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
} from '@/entities/notifiers';
import { isNotifierAddressChanged } from '@/features/edit-notifier/lib/isNotifierAddressChanged';

// A saved notifier may leave its secrets empty, because the backend keeps the stored ones, except
// those a changed resource address must get again.
export const isNotifierSettingsFilled = (
  settings: NotifierSettings,
  isSaved: boolean,
  savedSettings?: NotifierSettings,
): boolean => {
  const isCreate = !isSaved;
  const isAddressChanged = isNotifierAddressChanged(settings, savedSettings);

  switch (settings.type) {
    case NotifierType.EMAIL:
      return validateEmailNotifier(settings, isAddressChanged);
    case NotifierType.TELEGRAM:
      return validateTelegramNotifier(isCreate, settings, isAddressChanged);
    case NotifierType.SLACK:
      return validateSlackNotifier(isCreate, settings);
    case NotifierType.DISCORD:
      return validateDiscordNotifier(isCreate, settings);
    case NotifierType.TEAMS:
      return validateTeamsNotifier(isCreate, settings);
    case NotifierType.MATTERMOST:
      return validateMattermostNotifier(isCreate, settings, isAddressChanged);
    case NotifierType.WEBHOOK:
      return validateWebhookNotifier(
        settings,
        isAddressChanged && savedSettings?.type === NotifierType.WEBHOOK
          ? savedSettings.headers
          : [],
      );
  }
};
