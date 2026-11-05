import { describe, expect, it } from 'vitest';

import { EmailNotifierSecurity } from './EmailNotifierSecurity';
import { getDefaultEmailNotifierSecurity } from './getDefaultEmailNotifierSecurity';
import { getSecurityAfterPortChange } from './getSecurityAfterPortChange';

describe('getDefaultEmailNotifierSecurity', () => {
  it('returns tls only for port 465', () => {
    expect(getDefaultEmailNotifierSecurity(465)).toBe(EmailNotifierSecurity.TLS);
    expect(getDefaultEmailNotifierSecurity(587)).toBe(EmailNotifierSecurity.STARTTLS);
    expect(getDefaultEmailNotifierSecurity(25)).toBe(EmailNotifierSecurity.STARTTLS);
    expect(getDefaultEmailNotifierSecurity(0)).toBe(EmailNotifierSecurity.STARTTLS);
  });
});

describe('getSecurityAfterPortChange', () => {
  it('follows the new port when the mode is the previous port default', () => {
    expect(getSecurityAfterPortChange(EmailNotifierSecurity.STARTTLS, 587, 465)).toBe(
      EmailNotifierSecurity.TLS,
    );
    expect(getSecurityAfterPortChange(EmailNotifierSecurity.TLS, 465, 587)).toBe(
      EmailNotifierSecurity.STARTTLS,
    );
  });

  it('keeps a mode that differs from the previous port default', () => {
    expect(getSecurityAfterPortChange(EmailNotifierSecurity.NONE, 25, 2525)).toBe(
      EmailNotifierSecurity.NONE,
    );
    expect(getSecurityAfterPortChange(EmailNotifierSecurity.TLS, 587, 2525)).toBe(
      EmailNotifierSecurity.TLS,
    );
  });
});
