import { useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useCallback, useEffect, useEffectEvent, useState } from 'react';

import { useSessionQuery } from '@/shared/api';

interface Options<Item> {
  queryKey: QueryKey;
  read: (signal: AbortSignal) => Promise<Item[]>;
  onError: (error: unknown) => void;
}

export const useResourceCollection = <Item extends { id: string }>(options: Options<Item>) => {
  const client = useQueryClient();
  const query = useSessionQuery({ queryKey: options.queryKey, queryFn: options.read });
  const [isCleared, setIsCleared] = useState(false);
  const reportFailure = useEffectEvent(options.onError);

  useEffect(() => {
    if (query.error !== null) reportFailure(query.error);
  }, [query.error, query.errorUpdatedAt]);

  useEffect(() => {
    if (query.data !== undefined) setIsCleared(false);
  }, [query.data, query.dataUpdatedAt]);

  const { refetch } = query;
  const reload = useCallback(
    async (isSilent = false) => {
      await refetch({ cancelRefetch: !isSilent });
    },
    [refetch],
  );
  const reconcile = (update: (items: Item[]) => Item[]) => {
    void client.cancelQueries({ queryKey: query.queryKey, exact: true });
    client.setQueryData<Item[]>(query.queryKey, (items = []) => update(items));
  };

  return {
    items: isCleared ? [] : (query.data ?? []),
    isReady: !isCleared && query.data !== undefined,
    loadError: query.error ?? undefined,
    isLoading: query.isPending,
    isSessionActive: query.isSessionActive,
    reload,
    clear: () => setIsCleared(true),
    update: reconcile,
    accept: (item: Item) =>
      reconcile((items) => [...items.filter((resource) => resource.id !== item.id), item]),
    remove: (itemId: string) => reconcile((items) => items.filter((item) => item.id !== itemId)),
  };
};
