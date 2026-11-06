import type { TeamsNotifierSettings } from '@/entities/notifiers/model/teams/TeamsNotifierSettings';

// A saved notifier keeps its stored URL when the form leaves it empty.
export const validateTeamsNotifier = (
  isCreate: boolean,
  settings: TeamsNotifierSettings,
): boolean => {
  if (!settings.powerAutomateUrl) {
    return !isCreate;
  }

  try {
    const powerAutomateUrl = new URL(settings.powerAutomateUrl);

    return powerAutomateUrl.protocol === 'http:' || powerAutomateUrl.protocol === 'https:';
  } catch {
    return false;
  }
};
