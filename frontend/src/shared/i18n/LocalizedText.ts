import type { TFunction } from 'i18next';

import type { TranslationKey } from '@/shared/i18n/TranslationKey';

// Returned by code outside components that needs to hand a message with parameters to the UI.
// It carries the key, not the text, so the component translates it in the current language on
// every render.
export interface LocalizedText {
  key: TranslationKey;
  params?: Record<string, string | number>;
}

export const translateLocalizedText = (text: LocalizedText, t: TFunction): string =>
  t(text.key, text.params ?? {});
