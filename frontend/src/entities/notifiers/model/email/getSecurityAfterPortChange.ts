import type { EmailNotifierSecurity } from './EmailNotifierSecurity';
import { getDefaultEmailNotifierSecurity } from './getDefaultEmailNotifierSecurity';

// A mode that differs from the previous port's default was chosen on purpose, whether in this
// session or before the channel was saved, so a port change must not overwrite it.
export const getSecurityAfterPortChange = (
  currentSecurity: EmailNotifierSecurity,
  previousPort: number,
  nextPort: number,
): EmailNotifierSecurity =>
  currentSecurity === getDefaultEmailNotifierSecurity(previousPort)
    ? getDefaultEmailNotifierSecurity(nextPort)
    : currentSecurity;
