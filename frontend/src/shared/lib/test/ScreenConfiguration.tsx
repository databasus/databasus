import type { ReactNode } from 'react';

import { ControlLocalization } from '@/shared/i18n';
import { FeedbackProvider } from '@/shared/lib/feedback';
import { ThemeConfiguration } from '@/shared/theme';

export const ScreenConfiguration = ({ children }: { children: ReactNode }) => {
  return (
    <ThemeConfiguration isTestEnvironment>
      <FeedbackProvider>
        <ControlLocalization>{children}</ControlLocalization>
      </FeedbackProvider>
    </ThemeConfiguration>
  );
};
