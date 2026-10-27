import { type RefObject, useEffect, useState } from 'react';

import {
  PagedListController,
  type PagedListOptions,
  type PagedListState,
  createInitialPagedListState,
} from '../lib/PagedListController';

const SCROLL_LOAD_THRESHOLD_PX = 100;

export interface PagedListParams<TItem, TCursor> extends PagedListOptions<TItem, TCursor> {
  scrollContainerRef?: RefObject<HTMLDivElement | null>;
  // A change discards the loaded rows and loads the first page again.
  resetKey: string;
}

// Pages a list as the user scrolls: the next page loads when the container is
// scrolled near its bottom.
export function usePagedList<TItem, TCursor>({
  fetchPage,
  getItemKey,
  onError,
  scrollContainerRef,
  resetKey,
}: PagedListParams<TItem, TCursor>) {
  const [state, setState] = useState<PagedListState<TItem, TCursor>>(createInitialPagedListState);
  const [controller] = useState(
    () => new PagedListController<TItem, TCursor>({ fetchPage, getItemKey, onError }, setState),
  );

  // The scroll listener and the caller's timers outlive a render, so the
  // controller reads the latest callbacks. Declared first so it runs before the reload.
  useEffect(() => {
    controller.setOptions({ fetchPage, getItemKey, onError });
  });

  useEffect(() => {
    controller.reload();
  }, [resetKey]);

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
  }, [scrollContainerRef]);

  return {
    items: state.list.items,
    isLoading: state.isLoading,
    isLoadingMore: state.isLoadingMore,
    hasMore: state.list.nextCursor !== undefined,
    reload: () => controller.reload(),
    refreshFirstPage: () => controller.refreshFirstPage(),
    replaceOrPrependItem: (item: TItem) => controller.replaceOrPrependItem(item),
  };
}
