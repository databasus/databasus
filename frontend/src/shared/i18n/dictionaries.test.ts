import { describe, expect, it } from 'vitest';

import { DICTIONARIES } from './dictionaries';

type Dictionary = { [key: string]: string | Dictionary };

const flatten = (dictionary: Dictionary, prefix = ''): Map<string, string> => {
  const entries = new Map<string, string>();

  for (const [key, value] of Object.entries(dictionary)) {
    const path = prefix ? `${prefix}.${key}` : key;

    if (typeof value === 'string') {
      entries.set(path, value);
    } else {
      flatten(value, path).forEach((nestedValue, nestedPath) =>
        entries.set(nestedPath, nestedValue),
      );
    }
  }

  return entries;
};

const uniqueSorted = (values: string[]) => [...new Set(values)].sort();

// {{name}} and {{name, format}} both count as the placeholder "name"
const getPlaceholders = (text: string) =>
  uniqueSorted([...text.matchAll(/\{\{\s*([^}\s,]+)[^}]*\}\}/g)].map((match) => match[1]));

// <bold>, </bold> and <workspaceName/> all count as the Trans tag of that name
const getTransTags = (text: string) =>
  uniqueSorted([...text.matchAll(/<\/?([A-Za-z][\w-]*)\s*\/?>/g)].map((match) => match[1]));

// Commands and identifiers inside <code> are technical text and stay byte-identical
const getCodeSpans = (text: string) =>
  [...text.matchAll(/<code>(.*?)<\/code>/g)].map((match) => match[1]).sort();

const english = flatten(DICTIONARIES.en);

const translations = Object.entries(DICTIONARIES).filter(([locale]) => locale !== 'en');

describe.each(translations)('the %s dictionary', (locale, dictionary) => {
  const translated = flatten(dictionary);

  // The typeof en annotation catches a missing or leftover key at compile time, but a dictionary
  // built through a variable or a spread escapes that check. Comparing the flat key sets keeps the
  // runtime shape honest for every shipped language.
  it('has exactly the English key set', () => {
    expect([...translated.keys()].sort()).toEqual([...english.keys()].sort());
  });

  it('uses the same {{placeholders}} as English in every message', () => {
    const mismatches = [...english]
      .filter(([key, text]) => {
        const translatedText = translated.get(key) ?? '';
        return getPlaceholders(translatedText).join() !== getPlaceholders(text).join();
      })
      .map(([key, text]) => `${locale}: ${key}: "${translated.get(key)}" vs en "${text}"`);

    expect(mismatches).toEqual([]);
  });

  it('uses the same Trans tags as English in every message', () => {
    const mismatches = [...english]
      .filter(([key, text]) => {
        const translatedText = translated.get(key) ?? '';
        return getTransTags(translatedText).join() !== getTransTags(text).join();
      })
      .map(([key, text]) => `${locale}: ${key}: "${translated.get(key)}" vs en "${text}"`);

    expect(mismatches).toEqual([]);
  });

  it('keeps every <code> span identical to English', () => {
    const mismatches = [...english]
      .filter(([key, text]) => {
        const translatedText = translated.get(key) ?? '';
        return getCodeSpans(translatedText).join('|') !== getCodeSpans(text).join('|');
      })
      .map(([key]) => `${locale}: ${key}`);

    expect(mismatches).toEqual([]);
  });

  it('has no empty message', () => {
    const emptyKeys = [...translated].filter(([, text]) => text.trim() === '').map(([key]) => key);

    expect(emptyKeys).toEqual([]);
  });
});
