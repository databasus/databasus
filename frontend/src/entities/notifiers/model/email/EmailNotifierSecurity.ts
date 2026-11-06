import type { ApiComponents } from '@/shared/api';

export type EmailNotifierSecurity = ApiComponents['schemas']['SmtpSecurity'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const EmailNotifierSecurity = {
  TLS: 'TLS',
  STARTTLS: 'STARTTLS',
  NONE: 'NONE',
} as const satisfies Record<string, EmailNotifierSecurity>;
