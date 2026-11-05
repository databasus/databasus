import { en } from './locales/en';
import { es } from './locales/es';
import { fr } from './locales/fr';
import { pt } from './locales/pt';
import { ru } from './locales/ru';
import { zh } from './locales/zh';

// The supported languages are exactly the dictionaries that exist. Adding a language means adding
// its dictionary here and its settings in localeSettings.ts.
export const DICTIONARIES = { en, ru, es, pt, zh, fr };

export type Locale = keyof typeof DICTIONARIES;

export const SUPPORTED_LOCALES = Object.keys(DICTIONARIES) as Locale[];

export const DEFAULT_LOCALE: Locale = 'en';
