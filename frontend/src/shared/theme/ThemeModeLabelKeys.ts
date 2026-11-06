import type { TranslationKey } from '@/shared/i18n';
import type { ThemeMode } from '@/shared/theme/themeContext';

export const THEME_MODE_LABEL_KEYS: Record<ThemeMode, TranslationKey> = {
  light: 'app.theme.light',
  dark: 'app.theme.dark',
  system: 'app.theme.system',
};
