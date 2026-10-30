export interface PagedListPage<TItem, TCursor, TMetadata = undefined> {
  items: TItem[];
  nextCursor?: TCursor;
  metadata?: TMetadata;
}

// Rows already held win over the page's copies: an offset page requested after a
// refresh added rows at the top starts early and repeats rows the list shows.
export const appendNextPage = <TItem, TCursor, TMetadata = undefined>(
  loadedList: PagedListPage<TItem, TCursor, TMetadata>,
  nextPage: PagedListPage<TItem, TCursor, TMetadata>,
  getItemKey: (item: TItem) => string,
): PagedListPage<TItem, TCursor, TMetadata> => {
  const loadedKeys = new Set(loadedList.items.map(getItemKey));
  const newItems = nextPage.items.filter((item) => !loadedKeys.has(getItemKey(item)));

  return {
    ...loadedList,
    items: [...loadedList.items, ...newItems],
    nextCursor: nextPage.nextCursor,
  };
};

// The refreshed first page replaces every loaded row up to its own last row, so
// rows that changed, moved or vanished near the top are corrected, while the
// tail loaded by scrolling stays. When that last row is not among the loaded
// ones, the loaded rows can no longer be lined up with the fresh page, and the
// list falls back to the first page alone.
export const mergeRefreshedFirstPage = <TItem, TCursor, TMetadata = undefined>(
  loadedList: PagedListPage<TItem, TCursor, TMetadata>,
  firstPage: PagedListPage<TItem, TCursor, TMetadata>,
  getItemKey: (item: TItem) => string,
): PagedListPage<TItem, TCursor, TMetadata> => {
  const lastFirstPageItem = firstPage.items[firstPage.items.length - 1];
  if (firstPage.nextCursor === undefined || lastFirstPageItem === undefined) {
    return firstPage;
  }

  const lastFirstPageKey = getItemKey(lastFirstPageItem);
  const lastFirstPageIndex = loadedList.items.findIndex(
    (item) => getItemKey(item) === lastFirstPageKey,
  );

  const isLoadedTailEmpty =
    lastFirstPageIndex === -1 || lastFirstPageIndex === loadedList.items.length - 1;
  if (isLoadedTailEmpty) {
    return firstPage;
  }

  const firstPageKeys = new Set(firstPage.items.map(getItemKey));
  const loadedTail = loadedList.items
    .slice(lastFirstPageIndex + 1)
    .filter((item) => !firstPageKeys.has(getItemKey(item)));

  return {
    ...firstPage,
    items: [...firstPage.items, ...loadedTail],
    nextCursor: loadedList.nextCursor,
  };
};
