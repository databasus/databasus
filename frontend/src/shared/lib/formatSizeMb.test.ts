import { describe, expect, it } from 'vitest';

import { formatSizeMb } from './formatSizeMb';

const formatNumber = (value: number) => new Intl.NumberFormat('en').format(value);

describe('formatSizeMb', () => {
  it('shows sizes below one gigabyte in megabytes', () => {
    expect(formatSizeMb(10.5, formatNumber)).toBe('10.5 MB');
  });

  it('rounds megabytes to two decimals', () => {
    expect(formatSizeMb(0.12345, formatNumber)).toBe('0.12 MB');
  });

  it('switches to gigabytes at exactly 1024 megabytes', () => {
    expect(formatSizeMb(1024, formatNumber)).toBe('1 GB');
  });

  it('rounds gigabytes to two decimals and groups digits', () => {
    expect(formatSizeMb(1536000, formatNumber)).toBe('1,500 GB');
  });

  it('treats a missing size as zero', () => {
    expect(formatSizeMb(undefined, formatNumber)).toBe('0 MB');
  });
});
