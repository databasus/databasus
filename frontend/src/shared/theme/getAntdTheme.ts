import { theme, type ThemeConfig } from 'antd';

import type { ResolvedTheme } from './themeContext';

export const getAntdTheme = (resolvedTheme: ResolvedTheme): ThemeConfig => ({
  algorithm: resolvedTheme === 'dark' ? theme.darkAlgorithm : theme.defaultAlgorithm,
  cssVar: true,
  token: {
    colorPrimary: '#155dfc',
    colorSuccess: '#00c950',
    ...(resolvedTheme === 'dark'
      ? {
          colorTextDescription: '#bfbfbf',
          colorTextPlaceholder: '#a6a6a6',
          colorBorder: '#424242',
          colorLink: '#91caff',
          colorLinkHover: '#bae0ff',
          colorLinkActive: '#69b1ff',
          colorErrorText: '#ff7875',
          colorSuccessText: '#05df72',
          colorWarningText: '#ffd666',
        }
      : {
          colorTextDescription: '#595959',
          colorTextPlaceholder: '#737373',
          colorBorder: '#d9d9d9',
          controlItemBgActive: '#dbeafe',
          colorLink: '#155dfc',
          colorLinkHover: '#0042d6',
          colorLinkActive: '#0033a5',
          colorErrorText: '#b42318',
          colorSuccessText: '#007a33',
          colorWarningText: '#875800',
        }),
    // eslint-disable-next-line i18next/no-literal-string -- CSS font-family stack
    fontFamily: 'Jost, system-ui, -apple-system, "Segoe UI", "Noto Sans CJK SC", sans-serif',
  },
});
