import { App, ConfigProvider } from 'antd';
import type { ReactNode } from 'react';

import { useLocale } from '../../i18n';
import { useTheme, getAntdTheme } from '../../theme';

export const ScreenConfiguration = ({ children }: { children: ReactNode }) => {
  const { antdLocale } = useLocale();
  const { resolvedTheme } = useTheme();
  return (
    <ConfigProvider locale={antdLocale} theme={getAntdTheme(resolvedTheme)}>
      <App>{children}</App>
    </ConfigProvider>
  );
};
