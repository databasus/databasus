import { describe, expect, it } from 'vitest';

import { SUPPORTED_LOCALES } from './dictionaries';
import { resolveInitialLocale } from './resolveInitialLocale';

const ALL_LOCALES = ['en', 'ru', 'es', 'pt', 'zh', 'fr', 'id'];

describe('resolveInitialLocale', () => {
  it('takes the highest-ranked supported browser language', () => {
    expect(
      resolveInitialLocale({
        storedLocale: null,
        browserLanguages: ['de', 'ru'],
        supportedLocales: ['en', 'ru'],
      }),
    ).toBe('ru');
  });

  it('falls back to English when no browser language is supported', () => {
    expect(
      resolveInitialLocale({
        storedLocale: null,
        browserLanguages: ['de-DE', 'it'],
        supportedLocales: ['en', 'ru'],
      }),
    ).toBe('en');
  });

  it('falls back to English for an empty browser list', () => {
    expect(
      resolveInitialLocale({ storedLocale: null, browserLanguages: [], supportedLocales: ['en'] }),
    ).toBe('en');
  });

  it('falls back to English when Indonesian is not a supported locale', () => {
    expect(
      resolveInitialLocale({
        storedLocale: null,
        browserLanguages: ['id-ID'],
        supportedLocales: ['en', 'ru'],
      }),
    ).toBe('en');
  });

  it('resolves id-ID to Indonesian from the supported locales', () => {
    expect(
      resolveInitialLocale({
        storedLocale: null,
        browserLanguages: ['id-ID'],
        supportedLocales: SUPPORTED_LOCALES,
      }),
    ).toBe('id');
  });

  it.each([
    ['zh-TW', 'zh'],
    ['zh-HK', 'zh'],
    ['zh-CN', 'zh'],
    ['pt-PT', 'pt'],
    ['pt-BR', 'pt'],
    ['es-419', 'es'],
    ['FR-ca', 'fr'],
    ['id-ID', 'id'],
  ])('maps the regional variant %s to %s', (browserLanguage, resolvedLocale) => {
    expect(
      resolveInitialLocale({
        storedLocale: null,
        browserLanguages: [browserLanguage],
        supportedLocales: ALL_LOCALES,
      }),
    ).toBe(resolvedLocale);
  });

  it('lets a stored supported choice outrank the browser preference', () => {
    expect(
      resolveInitialLocale({
        storedLocale: 'en',
        browserLanguages: ['es'],
        supportedLocales: ALL_LOCALES,
      }),
    ).toBe('en');
  });

  it('keeps a stored Indonesian choice', () => {
    expect(
      resolveInitialLocale({
        storedLocale: 'id',
        browserLanguages: ['en'],
        supportedLocales: ALL_LOCALES,
      }),
    ).toBe('id');
  });

  it('ignores a stored value that is not supported', () => {
    expect(
      resolveInitialLocale({
        storedLocale: 'de',
        browserLanguages: ['ru'],
        supportedLocales: ['en', 'ru'],
      }),
    ).toBe('ru');
  });
});
