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
  dayjsLocale: string;
}

export const LOCALE_SETTINGS: Record<Locale, LocaleSettings> = {
  en: { languageTag: 'en', dayjsLocale: 'en' },
  ru: { languageTag: 'ru', dayjsLocale: 'ru' },
  es: { languageTag: 'es', dayjsLocale: 'es' },
  // Brazilian Portuguese is the one Portuguese variant shipped
  pt: { languageTag: 'pt-BR', dayjsLocale: 'pt-br' },
  // Simplified Chinese is the one Chinese variant shipped
  zh: { languageTag: 'zh-CN', dayjsLocale: 'zh-cn' },
  fr: { languageTag: 'fr', dayjsLocale: 'fr' },
  id: { languageTag: 'id-ID', dayjsLocale: 'id' },
};
