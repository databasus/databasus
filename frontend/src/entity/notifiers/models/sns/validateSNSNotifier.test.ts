import { describe, expect, it } from 'vitest';

import { validateSNSNotifier } from './validateSNSNotifier';

describe('validateSNSNotifier', () => {
  it('returns false when topicArn is missing', () => {
    expect(
      validateSNSNotifier({
        topicArn: '',
        region: 'us-east-1',
        authType: 'KEYS',
        accessKey: 'key',
        secretKey: 'secret',
      }),
    ).toBe(false);
  });

  it('returns false when region is missing', () => {
    expect(
      validateSNSNotifier({
        topicArn: 'arn:aws:sns:us-east-1:123456789012:topic',
        region: '',
        authType: 'KEYS',
        accessKey: 'key',
        secretKey: 'secret',
      }),
    ).toBe(false);
  });

  it('returns false in KEYS mode when keys are missing', () => {
    expect(
      validateSNSNotifier({
        topicArn: 'arn:aws:sns:us-east-1:123456789012:topic',
        region: 'us-east-1',
        authType: 'KEYS',
      }),
    ).toBe(false);
  });

  it('validates KEYS mode when keys are provided', () => {
    expect(
      validateSNSNotifier({
        topicArn: 'arn:aws:sns:us-east-1:123456789012:topic',
        region: 'us-east-1',
        authType: 'KEYS',
        accessKey: 'key',
        secretKey: 'secret',
      }),
    ).toBe(true);
  });

  it('validates IAM_ROLE mode without keys', () => {
    expect(
      validateSNSNotifier({
        topicArn: 'arn:aws:sns:us-east-1:123456789012:topic',
        region: 'us-east-1',
        authType: 'IAM_ROLE',
      }),
    ).toBe(true);
  });
});
