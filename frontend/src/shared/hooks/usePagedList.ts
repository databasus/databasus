import { type RefObject, useCallback, useEffect, useState } from 'react';

import {
  PagedListController,
  type PagedListOptions,
  type PagedListState,
  createInitialPagedListState,
} from '../lib/PagedListController';

const SCROLL_LOAD_THRESHOLD_PX = 100;

export interface PagedListParams<TItem, TCursor, TMetadata = undefined> extends PagedListOptions<
  TItem,
  TCursor,
  TMetadata
> {
  scrollContainerRef?: RefObject<HTMLDivElement | null>;
  // A change discards the loaded rows and loads the first page again.
  resetKey: string;
}

// Pages a list as the user scrolls: the next page loads when the container is
// scrolled near its bottom.
export function usePagedList<TItem, TCursor, TMetadata = undefined>({
  fetchPage,
  getItemKey,
  onError,
  scrollContainerRef,
  resetKey,
}: PagedListParams<TItem, TCursor, TMetadata>) {
  const [state, setState] = useState<PagedListState<TItem, TCursor, TMetadata>>(
    createInitialPagedListState,
  );
  const [controller] = useState(
    () =>
      new PagedListController<TItem, TCursor, TMetadata>(
        { fetchPage, getItemKey, onError },
        setState,
      ),
  );

  useEffect(() => {
    controller.setOptions({ fetchPage, getItemKey, onError });
  }, [controller, fetchPage, getItemKey, onError]);

  useEffect(() => {
    controller.activate();
    void controller.reload();
    return () => controller.deactivate();
  }, [controller, resetKey]);

  useEffect(() => {
    const container = scrollContainerRef?.current;
    if (!container) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container;

      if (scrollHeight - scrollTop <= clientHeight + SCROLL_LOAD_THRESHOLD_PX) {
        controller.loadNextPage();
      }
    };

    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [controller, scrollContainerRef]);

  const reload = useCallback(() => controller.reload(), [controller]);
  const refreshFirstPage = useCallback(() => controller.refreshFirstPage(), [controller]);
  const replaceOrPrependItem = useCallback(
    (item: TItem) => controller.replaceOrPrependItem(item),
    [controller],
  );

  return {
    items: state.list.items,
    metadata: state.list.metadata,
    isLoading: state.isLoading,
    isLoadingMore: state.isLoadingMore,
    firstLoadError: state.firstLoadError,
    hasMore: state.list.nextCursor !== undefined,
    reload,
    refreshFirstPage,
    replaceOrPrependItem,
  };
}
