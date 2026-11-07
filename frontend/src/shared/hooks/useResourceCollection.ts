import { useCallback, useEffect, useState } from 'react';

import { useSessionGuard } from '@/shared/auth';
import { useLatestRef } from '@/shared/hooks/useLatestRef';
import { useRequestRevision } from '@/shared/hooks/useRequestRevision';

interface Options<Item> {
  scopeKey: string;
  read: () => Promise<Item[]>;
  onError: (error: unknown) => void;
}

export const useResourceCollection = <Item extends { id: string }>(options: Options<Item>) => {
  const [collection, setCollection] = useState<{ items: Item[]; isReady: boolean }>({
    items: [],
    isReady: false,
  });
  const [loadError, setLoadError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const isCurrent = useSessionGuard();
  const revision = useRequestRevision();
  const latestOptions = useLatestRef(options);

  useEffect(() => {
    setCollection({ items: [], isReady: false });
    setLoadError(undefined);
    setIsLoading(true);

    return () => {
      revision.Invalidate();
    };
  }, [options.scopeKey, revision]);

  const reload = useCallback(
    async (isSilent = false) => {
      if (!isCurrent()) return;

      const requestRevision = revision.Invalidate();
      const isActive = () => isCurrent() && revision.IsCurrent(requestRevision);

      if (!isSilent) setIsLoading(true);

      setLoadError(undefined);

      try {
        const items = await latestOptions.current.read();
        if (isActive()) setCollection({ items, isReady: true });
      } catch (error) {
        if (!isActive()) return;

        setLoadError(error);
        latestOptions.current.onError(error);
      } finally {
        if (isActive()) setIsLoading(false);
      }
    },
    [isCurrent, latestOptions, revision],
  );

  const clear = () => setCollection({ items: [], isReady: false });
  const accept = (item: Item) =>
    setCollection((current) => ({
      isReady: true,
      items: [...current.items.filter((resource) => resource.id !== item.id), item],
    }));
  const remove = (itemId: string) =>
    setCollection((current) => ({
      ...current,
      items: current.items.filter((item) => item.id !== itemId),
    }));

  return { ...collection, loadError, isLoading, reload, clear, accept, remove };
};
