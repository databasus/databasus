import { theme, type ThemeConfig } from 'antd';

import type { ResolvedTheme } from './themeContext';

export const getAntdTheme = (resolvedTheme: ResolvedTheme): ThemeConfig => ({
  algorithm: resolvedTheme === 'dark' ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: { colorPrimary: '#155dfc' },
});
