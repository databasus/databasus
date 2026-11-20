/* eslint-disable i18next/no-literal-string -- locale codes */
import 'dayjs/locale/es';
import 'dayjs/locale/fr';
import 'dayjs/locale/id';
import 'dayjs/locale/pt-br';
import 'dayjs/locale/ru';
import 'dayjs/locale/zh-cn';
import type { Locale } from '@/shared/i18n/dictionaries';

interface LocaleSettings {
  // BCP 47 tag for the document's lang attribute and for Intl number formatting
  languageTag: string;
  // BCP 47 tag for dates; English writes the day before the month, as British English does
  dateLanguageTag: string;
  dayjsLocale: string;
}

export const LOCALE_SETTINGS: Record<Locale, LocaleSettings> = {
  en: { languageTag: 'en', dateLanguageTag: 'en-GB', dayjsLocale: 'en' },
  ru: { languageTag: 'ru', dateLanguageTag: 'ru', dayjsLocale: 'ru' },
  es: { languageTag: 'es', dateLanguageTag: 'es', dayjsLocale: 'es' },
  // Brazilian Portuguese is the one Portuguese variant shipped
  pt: { languageTag: 'pt-BR', dateLanguageTag: 'pt-BR', dayjsLocale: 'pt-br' },
  // Simplified Chinese is the one Chinese variant shipped
  zh: { languageTag: 'zh-CN', dateLanguageTag: 'zh-CN', dayjsLocale: 'zh-cn' },
  fr: { languageTag: 'fr', dateLanguageTag: 'fr', dayjsLocale: 'fr' },
  id: { languageTag: 'id-ID', dateLanguageTag: 'id-ID', dayjsLocale: 'id' },
};
