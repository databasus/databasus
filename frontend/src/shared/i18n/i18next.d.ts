import 'i18next';
import type { en } from '@/shared/i18n/locales/en';

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'translation';
    nsSeparator: false;
    resources: {
      translation: typeof en;
    };
  }
}
