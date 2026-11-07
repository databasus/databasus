import { MantineProvider } from '@mantine/core';
import type { ReactNode } from 'react';

import { mantineCssVariables, mantineTheme } from '@/shared/theme/mantineTheme';
import { ThemeCssVariables } from '@/shared/theme/ThemeCssVariables';
import { useTheme } from '@/shared/theme/useTheme';

interface Props {
  children: ReactNode;
  isTestEnvironment?: boolean;
}

export const ThemeConfiguration = ({ children, isTestEnvironment = false }: Props) => {
  const { resolvedTheme } = useTheme();

  return (
    <MantineProvider
      theme={mantineTheme}
      cssVariablesResolver={mantineCssVariables}
      forceColorScheme={resolvedTheme}
      env={isTestEnvironment ? 'test' : 'default'}
    >
      <ThemeCssVariables />
      {children}
    </MantineProvider>
  );
};
