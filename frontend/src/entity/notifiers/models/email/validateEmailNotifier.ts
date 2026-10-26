import type { EmailNotifierSettings } from './EmailNotifierSettings';

export const validateEmailNotifier = (settings: EmailNotifierSettings): boolean =>
  !!settings.targetEmail && !!settings.smtpHost && !!settings.smtpPort;
