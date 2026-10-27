import dayjs from 'dayjs';
import 'dayjs/locale/id';
import 'dayjs/locale/ru';
import { describe, expect, it } from 'vitest';

import { formatNumber } from './formatNumber';
import { formatRelativeTime } from './formatRelativeTime';

// Intl uses U+202F (narrow no-break space) or U+00A0 as the group separator depending on the
// language and ICU version; the tests compare with any whitespace normalized to a plain space.
const normalizeSpaces = (value: string) => value.replace(/\s/g, ' ');

describe('formatNumber', () => {
  it('groups digits by the selected language, not by the environment', () => {
    expect(normalizeSpaces(formatNumber(1_800_000, 'fr'))).toBe('1 800 000');
    expect(formatNumber(1_800_000, 'en')).toBe('1,800,000');
  });

  it('keeps full Western grouping for Chinese instead of the 万 form', () => {
    expect(formatNumber(1_800_000, 'zh-CN')).toBe('1,800,000');
  });

  it('groups Spanish, Portuguese and Russian the way each language does', () => {
    expect(formatNumber(1_800_000, 'es')).toBe('1.800.000');
    expect(formatNumber(1_800_000, 'pt-BR')).toBe('1.800.000');
    expect(normalizeSpaces(formatNumber(1_800_000, 'ru'))).toBe('1 800 000');
  });

  it('groups Indonesian thousands with points and writes a comma decimal', () => {
    expect(formatNumber(1_800_000, 'id-ID')).toBe('1.800.000');
    expect(formatNumber(1234.56, 'id-ID')).toBe('1.234,56');
  });

  it('formats only the number, leaving unit words to the caller', () => {
    expect(formatNumber(1234.56, 'en')).toBe('1,234.56');
    expect(formatNumber(1234.56, 'ru')).toMatch(/^1\s234,56$/);
  });
});

describe('formatRelativeTime', () => {
  it('writes a relative time in the selected language', () => {
    const twoHoursAgo = dayjs().subtract(2, 'hour');

    expect(formatRelativeTime(twoHoursAgo, 'ru')).toMatch(
      /^2 \p{Script=Cyrillic}+ \p{Script=Cyrillic}+$/u,
    );
    expect(formatRelativeTime(twoHoursAgo, 'en')).toBe('2 hours ago');
  });

  it('writes an Indonesian relative time without changing the global dayjs locale', () => {
    const twoHoursAgo = dayjs().subtract(2, 'hour');

    expect(formatRelativeTime(twoHoursAgo, 'id')).toBe('2 jam yang lalu');
    expect(dayjs.locale()).toBe('en');
  });

  it('does not change the global dayjs locale', () => {
    formatRelativeTime(dayjs(), 'ru');
    formatRelativeTime(dayjs(), 'id');

    expect(dayjs.locale()).toBe('en');
  });
});
