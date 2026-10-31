import { theme, type ThemeConfig } from 'antd';

import type { ResolvedTheme } from './themeContext';

export const getAntdTheme = (resolvedTheme: ResolvedTheme): ThemeConfig => ({
  algorithm: resolvedTheme === 'dark' ? theme.darkAlgorithm : theme.defaultAlgorithm,
  cssVar: true,
  token: {
    colorPrimary: '#155dfc',
    ...(resolvedTheme === 'dark'
      ? {
          colorTextDescription: '#bfbfbf',
          colorTextPlaceholder: '#a6a6a6',
          colorBorder: '#737373',
          colorLink: '#91caff',
          colorLinkHover: '#bae0ff',
          colorLinkActive: '#69b1ff',
          colorErrorText: '#ff7875',
          colorSuccessText: '#95de64',
          colorWarningText: '#ffd666',
        }
      : {
          colorTextDescription: '#595959',
          colorTextPlaceholder: '#737373',
          colorBorder: '#858585',
          colorLink: '#155dfc',
          colorLinkHover: '#0042d6',
          colorLinkActive: '#0033a5',
          colorErrorText: '#b42318',
          colorSuccessText: '#237804',
          colorWarningText: '#875800',
        }),
    // eslint-disable-next-line i18next/no-literal-string -- CSS font-family stack
    fontFamily: 'Jost, system-ui, -apple-system, "Segoe UI", "Noto Sans CJK SC", sans-serif',
  },
});
