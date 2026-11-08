import { QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState, type ReactNode } from 'react';

import { createQueryClient } from '@/shared/api';
import { subscribeSession } from '@/shared/auth';
import { ControlLocalization } from '@/shared/i18n';
import { FeedbackProvider } from '@/shared/lib/feedback';
import { ThemeConfiguration } from '@/shared/theme';

export const ScreenConfiguration = ({ children }: { children: ReactNode }) => {
  const [client] = useState(createQueryClient);

  useEffect(() => {
    const unsubscribe = subscribeSession(() => client.clear());

    return () => {
      unsubscribe();
      client.clear();
    };
  }, [client]);

  return (
    <ThemeConfiguration isTestEnvironment>
      <FeedbackProvider>
        <QueryClientProvider client={client}>
          <ControlLocalization>{children}</ControlLocalization>
        </QueryClientProvider>
      </FeedbackProvider>
    </ThemeConfiguration>
  );
};
