import i18next, { type i18n as I18nInstance } from 'i18next';
import { initReactI18next } from 'react-i18next';

import { DEFAULT_LOCALE, DICTIONARIES, type Locale } from '@/shared/i18n/dictionaries';

const resources = Object.fromEntries(
  Object.entries(DICTIONARIES).map(([locale, dictionary]) => [locale, { translation: dictionary }]),
);

// Resources are passed inline, so init() completes synchronously and the first render already has
// the chosen language's dictionary: no flash of English.
export const createI18n = (locale: Locale): I18nInstance => {
  const instance = i18next.createInstance();

  void instance.use(initReactI18next).init({
    lng: locale,
    // A last-resort guard only: completeness is enforced by `typeof en` at build time.
    fallbackLng: DEFAULT_LOCALE,
    resources,
    // Keys never contain a namespace prefix, so a colon inside a key must not be parsed as one.
    nsSeparator: false,
    interpolation: {
      // React escapes the string t() returns when it is rendered as a child.
      escapeValue: false,
    },
    react: {
      useSuspense: false,
      // Only the tags a component supplies through `components` become elements. A tag a
      // dictionary contains on its own, such as <strong>, is shown as text.
      transKeepBasicHtmlNodesFor: [],
      // Trans interpolates `values` before it parses the sentence for tags, so values are escaped
      // first and unescaped back into plain React text afterwards. Without this a value such as
      // "<strong>x</strong>" would become an element.
      transDefaultProps: {
        shouldUnescape: true,
        tOptions: { interpolation: { escapeValue: true } },
      },
    },
  });

  return instance;
};
