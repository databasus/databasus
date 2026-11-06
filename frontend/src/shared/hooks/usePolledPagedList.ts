import { useEffect } from 'react';

import { type PagedListParams, usePagedList } from '@/shared/hooks/usePagedList';

// eslint-disable-next-line i18next/no-literal-string -- DOM event name
const VISIBILITY_CHANGE_EVENT = 'visibilitychange';

interface Params<TItem, TCursor, TMetadata = undefined> extends PagedListParams<
  TItem,
  TCursor,
  TMetadata
> {
  refreshIntervalMs: number;
}

export function usePolledPagedList<TItem, TCursor, TMetadata = undefined>({
  refreshIntervalMs,
  ...pagedListParams
}: Params<TItem, TCursor, TMetadata>) {
  const pagedList = usePagedList(pagedListParams);

  const { refreshFirstPage } = pagedList;

  useEffect(() => {
    const refreshIfVisible = () => {
      if (document.visibilityState !== 'hidden') void refreshFirstPage();
    };

    const intervalId = setInterval(refreshIfVisible, refreshIntervalMs);
    document.addEventListener(VISIBILITY_CHANGE_EVENT, refreshIfVisible);

    return () => {
      clearInterval(intervalId);
      document.removeEventListener(VISIBILITY_CHANGE_EVENT, refreshIfVisible);
    };
  }, [refreshFirstPage, refreshIntervalMs]);

  return pagedList;
}
