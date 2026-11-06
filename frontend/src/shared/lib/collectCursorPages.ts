import type { PagedListPage } from '@/shared/lib/pagedList';

export const collectCursorPages = async <TItem, TCursor>(
  fetchPage: (cursor: TCursor | undefined) => Promise<PagedListPage<TItem, TCursor>>,
): Promise<TItem[]> => {
  const items: TItem[] = [];
  let cursor: TCursor | undefined;
  do {
    const page = await fetchPage(cursor);
    items.push(...page.items);
    cursor = page.nextCursor;
  } while (cursor !== undefined);
  return items;
};
