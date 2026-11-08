import {
  CancelledError,
  useQuery,
  type QueryKey,
  type UseQueryOptions,
  type RefetchOptions,
  useQueryClient,
  hashKey,
} from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';

import { getSessionSnapshot, isCurrentSession, useSession } from '@/shared/auth';

export const useSessionQuery = <Data>(
  options: Omit<UseQueryOptions<Data, unknown, Data, QueryKey>, 'queryFn'> & {
    queryFn: (signal: AbortSignal) => Promise<Data>;
    shouldFollowSession?: boolean;
  },
) => {
  const currentSession = useSession();
  const [initialSession] = useState(getSessionSnapshot);
  const { shouldFollowSession = false, ...queryOptions } = options;
  const session = shouldFollowSession ? currentSession : initialSession;
  const isSessionActive = isCurrentSession(session);
  const client = useQueryClient();
  const queryKey = ['session', session.generation, ...options.queryKey];
  const queryHash = hashKey(queryKey);
  const initialDataUpdateCount = useMemo(
    () => client.getQueryCache().get<Data>(queryHash)?.state.dataUpdateCount ?? 0,
    [client, queryHash],
  );
  const query = useQuery({
    ...queryOptions,
    enabled: isSessionActive ? options.enabled : false,
    queryKey,
    queryFn: async ({ signal }) => {
      if (!isCurrentSession(session)) throw new CancelledError();

      const data = await options.queryFn(signal);
      if (!isCurrentSession(session)) throw new CancelledError();

      return data;
    },
  });
  const { refetch } = query;
  const reload = useCallback(
    (options?: RefetchOptions) => {
      if (options?.cancelRefetch !== false)
        void client.cancelQueries({ predicate: (query) => query.queryHash === queryHash });
      return refetch(options);
    },
    [client, queryHash, refetch],
  );
  const dataRevision = client.getQueryCache().get<Data>(queryHash)?.state.dataUpdateCount ?? 0;
  const hasVisitData = dataRevision > initialDataUpdateCount;
  const isVisitPending = !isSessionActive || !query.isFetchedAfterMount;

  return {
    ...query,
    queryKey,
    dataRevision,
    isSessionActive,
    refetch: reload,
    data: isVisitPending || !hasVisitData ? undefined : query.data,
    error: isVisitPending ? null : query.error,
    isPending: isVisitPending || query.isPending,
    isLoading: (isVisitPending || query.isPending) && query.isFetching,
  };
};
