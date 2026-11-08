import type { ReadOptions } from '@/shared/api';
import type { PagedListPage } from '@/shared/lib/pagedList';

export const collectCursorPages = async <TItem, TCursor>(
  fetchPage: (cursor: TCursor | undefined) => Promise<PagedListPage<TItem, TCursor>>,
  { signal }: ReadOptions = {},
): Promise<TItem[]> => {
  const items: TItem[] = [];
  let cursor: TCursor | undefined;

  do {
    signal?.throwIfAborted();

    const page = await fetchPage(cursor);
    signal?.throwIfAborted();

    items.push(...page.items);
    cursor = page.nextCursor;
  } while (cursor !== undefined);

  return items;
};
