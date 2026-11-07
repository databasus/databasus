import type { ConfigType } from 'dayjs';
import { createContext } from 'react';

import type { Locale } from '@/shared/i18n/dictionaries';

export interface LocaleContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  dateLocale: string;
  formatNumber: (value: number, options?: Intl.NumberFormatOptions) => string;
  formatRelativeTime: (date: ConfigType) => string;
}

export const LocaleContext = createContext<LocaleContextValue | undefined>(undefined);
