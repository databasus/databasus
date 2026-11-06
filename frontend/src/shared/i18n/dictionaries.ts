import { en } from '@/shared/i18n/locales/en';
import { es } from '@/shared/i18n/locales/es';
import { fr } from '@/shared/i18n/locales/fr';
import { id } from '@/shared/i18n/locales/id';
import { pt } from '@/shared/i18n/locales/pt';
import { ru } from '@/shared/i18n/locales/ru';
import { zh } from '@/shared/i18n/locales/zh';

// The supported languages are exactly the dictionaries that exist. Adding a language means adding
// its dictionary here and its settings in localeSettings.ts.
export const DICTIONARIES = { en, ru, es, pt, zh, fr, id };

export type Locale = keyof typeof DICTIONARIES;

export const SUPPORTED_LOCALES = Object.keys(DICTIONARIES) as Locale[];

export const DEFAULT_LOCALE: Locale = 'en';
