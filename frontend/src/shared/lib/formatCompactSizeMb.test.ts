import { describe, expect, it } from 'vitest';

import { formatCompactSizeMb } from '@/shared/lib/formatCompactSizeMb';

const formatEnglishNumber = (value: number, options?: Intl.NumberFormatOptions) =>
  new Intl.NumberFormat('en', options).format(value);

describe('formatCompactSizeMb', () => {
  it.each([
    { sizeMb: 0, formattedSize: '0 MB' },
    { sizeMb: 104.4, formattedSize: '104 MB' },
    { sizeMb: 104.5, formattedSize: '105 MB' },
    { sizeMb: 1023.4, formattedSize: '1,023 MB' },
    { sizeMb: 1023.9, formattedSize: '1,024 MB' },
    { sizeMb: 1024, formattedSize: '1 GB' },
    { sizeMb: 10752, formattedSize: '10.5 GB' },
    { sizeMb: 10240, formattedSize: '10 GB' },
    { sizeMb: 10900, formattedSize: '10.6 GB' },
  ])('returns $formattedSize with $sizeMb recorded MB', ({ sizeMb, formattedSize }) => {
    expect(formatCompactSizeMb(sizeMb, formatEnglishNumber)).toBe(formattedSize);
  });
});
