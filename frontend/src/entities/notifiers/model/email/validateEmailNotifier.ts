import type { EmailNotifierSettings } from './EmailNotifierSettings';

export const validateEmailNotifier = (
  settings: EmailNotifierSettings,
  isAddressChanged = false,
): boolean =>
  /^.+@[^\s@]+$/.test(settings.targetEmail ?? '') &&
  !!settings.smtpHost?.trim() &&
  Number.isInteger(settings.smtpPort) &&
  settings.smtpPort >= 1 &&
  settings.smtpPort <= 65535 &&
  (!isAddressChanged || !settings.smtpUser || !!settings.smtpPassword);
