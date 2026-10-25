import { useEffect } from 'react';

import { type PagedListParams, usePagedList } from './usePagedList';

interface Params<TItem, TCursor> extends PagedListParams<TItem, TCursor> {
  refreshIntervalMs: number;
}

// Pages a newest-first list as the user scrolls and keeps it current: each tick
// re-requests only the first page and merges it into the loaded rows.
export function usePolledPagedList<TItem, TCursor>({
  refreshIntervalMs,
  ...pagedListParams
}: Params<TItem, TCursor>) {
  const pagedList = usePagedList(pagedListParams);

  useEffect(() => {
    const intervalId = setInterval(() => pagedList.refreshFirstPage(), refreshIntervalMs);

    return () => clearInterval(intervalId);
  }, [refreshIntervalMs]);

  return pagedList;
}
