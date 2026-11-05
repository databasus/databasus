import { describe, expect, it } from 'vitest';

import { validateSESNotifier } from './validateSESNotifier';

describe('validateSESNotifier', () => {
  it('returns false when fromEmail is missing', () => {
    expect(
      validateSESNotifier({
        fromEmail: '',
        targetEmail: 'admin@example.com',
        region: 'us-east-1',
        authType: 'IAM_ROLE',
      }),
    ).toBe(false);
  });

  it('returns false when targetEmail is missing', () => {
    expect(
      validateSESNotifier({
        fromEmail: 'sender@example.com',
        targetEmail: '',
        region: 'us-east-1',
        authType: 'IAM_ROLE',
      }),
    ).toBe(false);
  });

  it('returns false when region is missing', () => {
    expect(
      validateSESNotifier({
        fromEmail: 'sender@example.com',
        targetEmail: 'admin@example.com',
        region: '',
        authType: 'IAM_ROLE',
      }),
    ).toBe(false);
  });

  it('validates IAM_ROLE mode without keys', () => {
    expect(
      validateSESNotifier({
        fromEmail: 'sender@example.com',
        targetEmail: 'admin@example.com',
        region: 'us-east-1',
        authType: 'IAM_ROLE',
      }),
    ).toBe(true);
  });

  it('returns false in KEYS mode when keys are missing', () => {
    expect(
      validateSESNotifier({
        fromEmail: 'sender@example.com',
        targetEmail: 'admin@example.com',
        region: 'us-east-1',
        authType: 'KEYS',
      }),
    ).toBe(false);
  });

  it('validates KEYS mode when keys are provided', () => {
    expect(
      validateSESNotifier({
        fromEmail: 'sender@example.com',
        targetEmail: 'admin@example.com',
        region: 'us-east-1',
        authType: 'KEYS',
        accessKey: 'AKIAEXAMPLE',
        secretKey: 'secret',
      }),
    ).toBe(true);
  });
});
