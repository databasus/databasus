import { useEffect } from 'react';

import { type PagedListParams, usePagedList } from './usePagedList';

interface Params<TItem, TCursor, TMetadata = undefined> extends PagedListParams<
  TItem,
  TCursor,
  TMetadata
> {
  refreshIntervalMs: number;
}

// Pages a newest-first list as the user scrolls and keeps it current: each tick
// re-requests only the first page and merges it into the loaded rows.
export function usePolledPagedList<TItem, TCursor, TMetadata = undefined>({
  refreshIntervalMs,
  ...pagedListParams
}: Params<TItem, TCursor, TMetadata>) {
  const pagedList = usePagedList(pagedListParams);

  const { refreshFirstPage } = pagedList;

  useEffect(() => {
    const intervalId = setInterval(() => void refreshFirstPage(), refreshIntervalMs);

    return () => clearInterval(intervalId);
  }, [refreshFirstPage, refreshIntervalMs]);

  return pagedList;
}
