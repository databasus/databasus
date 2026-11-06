import { App, ConfigProvider } from 'antd';
import type { ReactNode } from 'react';

import { useLocale } from '@/shared/i18n';
import { useTheme, getAntdTheme, ThemeCssVariables } from '@/shared/theme';

export const ScreenConfiguration = ({ children }: { children: ReactNode }) => {
  const { antdLocale } = useLocale();
  const { resolvedTheme } = useTheme();
  return (
    <ConfigProvider
      button={{ autoInsertSpace: false }}
      locale={antdLocale}
      theme={getAntdTheme(resolvedTheme)}
    >
      <ThemeCssVariables />
      <App>{children}</App>
    </ConfigProvider>
  );
};
