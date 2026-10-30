import type { TranslationKey } from '../../../../shared/i18n';
import { EmailNotifierSecurity } from './EmailNotifierSecurity';

export const EMAIL_NOTIFIER_SECURITY_LABEL_KEYS: Record<EmailNotifierSecurity, TranslationKey> = {
  [EmailNotifierSecurity.TLS]: 'notifiers.email.securityModes.tls',
  [EmailNotifierSecurity.STARTTLS]: 'notifiers.email.securityModes.starttls',
  [EmailNotifierSecurity.NONE]: 'notifiers.email.securityModes.none',
};
