import { QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState, type ReactNode } from 'react';

import { createQueryClient } from '@/shared/api';
import { subscribeSession } from '@/shared/auth';

export const RequestConfiguration = ({ children }: { children: ReactNode }) => {
  const [client] = useState(createQueryClient);

  useEffect(() => {
    const unsubscribe = subscribeSession(() => client.clear());

    return () => {
      unsubscribe();
      client.clear();
    };
  }, [client]);

  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};
