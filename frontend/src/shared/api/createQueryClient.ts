import { notifyManager, QueryClient } from '@tanstack/react-query';

// Publish pending state before another user interaction can submit the same action.
notifyManager.setScheduler((notify) => notify());

export const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 0,
        gcTime: 0,
        refetchOnMount: 'always',
        retry: false,
        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
        networkMode: 'always',
      },
      mutations: { retry: false, gcTime: 0, networkMode: 'always' },
    },
  });
