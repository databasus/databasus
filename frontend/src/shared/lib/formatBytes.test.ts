import { describe, expect, it } from 'vitest';

import { formatNumber } from '@/shared/i18n/formatNumber';
import { formatBytes } from '@/shared/lib/formatBytes';

const formatEnglishNumber = (value: number, options?: Intl.NumberFormatOptions) =>
  formatNumber(value, 'en', options);

describe('formatBytes', () => {
  it.each([
    [0, '0 B'],
    [1023, '1,023 B'],
    [1024, '1 KB'],
    [1536, '1.5 KB'],
    [1024 ** 2 - 1, '1,024 KB'],
    [1024 ** 2, '1 MB'],
    [16 * 1024 ** 2, '16 MB'],
    [1024 ** 3, '1 GB'],
    [3.4 * 1024 ** 3, '3.4 GB'],
    [1024 ** 4, '1 TB'],
    [2048 * 1024 ** 4, '2,048 TB'],
  ])('formats %d bytes as %s', (bytes, expected) => {
    expect(formatBytes(bytes, formatEnglishNumber)).toBe(expected);
  });

  it('keeps the sign of a shrinking amount', () => {
    expect(formatBytes(-9 * 1024 ** 2, formatEnglishNumber)).toBe('-9 MB');
  });

  it('groups digits by the selected language', () => {
    const frenchBytes = formatBytes(1023, (value, options) => formatNumber(value, 'fr', options));

    expect(frenchBytes.replace(/\s/g, ' ')).toBe('1 023 B');
  });
});
