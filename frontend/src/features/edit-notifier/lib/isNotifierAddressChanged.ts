import { type NotifierSettings, NotifierType } from '@/entities/notifiers';
import { isOptionalValueChanged } from '@/shared/lib';

// Mirrors the resource address each notifier provider compares on the backend, which stays the
// authority: a stale list here only makes the user see the generic refusal.
export const isNotifierAddressChanged = (
  settings: NotifierSettings,
  savedSettings?: NotifierSettings,
): boolean => {
  if (!savedSettings) return false;

  switch (settings.type) {
    case NotifierType.EMAIL:
      return (
        savedSettings.type === NotifierType.EMAIL &&
        (isOptionalValueChanged(settings.smtpHost, savedSettings.smtpHost) ||
          isOptionalValueChanged(settings.smtpPort, savedSettings.smtpPort))
      );
    case NotifierType.TELEGRAM:
      // The API never returns the proxy URL, so a URL in the form is always a new one.
      return (
        savedSettings.type === NotifierType.TELEGRAM &&
        (isOptionalValueChanged(settings.botApiUrl, savedSettings.botApiUrl) ||
          settings.isProxyEnabled !== savedSettings.isProxyEnabled ||
          !!settings.proxyUrl)
      );
    case NotifierType.MATTERMOST:
      return (
        savedSettings.type === NotifierType.MATTERMOST &&
        isOptionalValueChanged(settings.serverUrl, savedSettings.serverUrl)
      );
    case NotifierType.WEBHOOK:
      return (
        savedSettings.type === NotifierType.WEBHOOK &&
        isOptionalValueChanged(settings.webhookUrl, savedSettings.webhookUrl)
      );
    default:
      return false;
  }
};
