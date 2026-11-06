import dayjs from 'dayjs';
import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { I18nextProvider } from 'react-i18next';

import { createI18n } from '@/shared/i18n/createI18n';
import { DEFAULT_LOCALE, type Locale, SUPPORTED_LOCALES } from '@/shared/i18n/dictionaries';
import { formatNumber } from '@/shared/i18n/formatNumber';
import { formatRelativeTime } from '@/shared/i18n/formatRelativeTime';
import { LocaleContext } from '@/shared/i18n/localeContext';
import { LOCALE_SETTINGS } from '@/shared/i18n/localeSettings';
import { resolveInitialLocale } from '@/shared/i18n/resolveInitialLocale';

// eslint-disable-next-line i18next/no-literal-string -- localStorage key
const LOCALE_STORAGE_KEY = 'databasus-locale';

function getInitialLocale(): Locale {
  if (typeof window === 'undefined') {
    return DEFAULT_LOCALE;
  }

  return resolveInitialLocale({
    storedLocale: localStorage.getItem(LOCALE_STORAGE_KEY),
    browserLanguages: navigator.languages ?? [navigator.language],
    supportedLocales: SUPPORTED_LOCALES,
  });
}

interface LocaleProviderProps {
  children: ReactNode;
}

export function LocaleProvider({ children }: LocaleProviderProps) {
  const [locale, setLocaleState] = useState<Locale>(getInitialLocale);

  // Created once with the resolved language, before the first render, so the first paint is
  // already in that language. dayjs is global, so its locale is set here for the same reason.
  const [i18n] = useState(() => {
    dayjs.locale(LOCALE_SETTINGS[locale].dayjsLocale);
    return createI18n(locale);
  });

  // The document is outside the React tree: its lang attribute (screen readers, the browser's
  // translate prompt) and its title follow the selection here. index.html keeps the English
  // values as what is shown before this runs.
  useEffect(() => {
    document.documentElement.lang = LOCALE_SETTINGS[locale].languageTag;
    document.title = i18n.t('app.documentTitle');
  }, [locale, i18n]);

  // The libraries switch before the new locale is published, so everything that re-renders from
  // the context or from useTranslation already reads the new language. The tree is never
  // remounted: open drawers and unsaved form input survive a switch.
  const setLocale = useCallback(
    (newLocale: Locale) => {
      dayjs.locale(LOCALE_SETTINGS[newLocale].dayjsLocale);
      void i18n.changeLanguage(newLocale);
      localStorage.setItem(LOCALE_STORAGE_KEY, newLocale);
      setLocaleState(newLocale);
    },
    [i18n],
  );

  const value = useMemo(() => {
    const { languageTag, dayjsLocale, antdLocale } = LOCALE_SETTINGS[locale];

    return {
      locale,
      setLocale,
      antdLocale,
      formatNumber: (number: number, options?: Intl.NumberFormatOptions) =>
        formatNumber(number, languageTag, options),
      formatRelativeTime: (date: dayjs.ConfigType) => formatRelativeTime(date, dayjsLocale),
    };
  }, [locale, setLocale]);

  return (
    <I18nextProvider i18n={i18n}>
      <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
    </I18nextProvider>
  );
}
