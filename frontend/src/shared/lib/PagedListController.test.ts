import { describe, expect, it, vi } from 'vitest';

import type { PagedListPage } from './pagedList';
import { PagedListController } from './PagedListController';

interface Row {
  id: string;
  status?: string;
}

type RowPage = PagedListPage<Row, string>;

const rows = (...ids: string[]): Row[] => ids.map((id) => ({ id }));

const createDeferredPage = () => {
  let resolvePage: (page: RowPage) => void = () => {};
  const promise = new Promise<RowPage>((resolve) => {
    resolvePage = resolve;
  });

  return { promise, resolvePage };
};

const createController = (fetchPage: (cursor: string | undefined) => Promise<RowPage>) => {
  const onError = vi.fn();
  const controller = new PagedListController<Row, string>(
    { fetchPage, getItemKey: (row) => row.id, onError },
    () => {},
  );

  return { controller, onError };
};

const getLoadedIds = (controller: PagedListController<Row, string>) =>
  controller.getState().list.items.map((row) => row.id);

describe('PagedListController', () => {
  it('loads the first page on reload', async () => {
    const fetchPage = vi.fn().mockResolvedValue({ items: rows('a', 'b'), nextCursor: 'after-b' });
    const { controller } = createController(fetchPage);

    await controller.reload();

    expect(fetchPage).toHaveBeenCalledWith(undefined);
    expect(getLoadedIds(controller)).toEqual(['a', 'b']);
    expect(controller.getState().isLoading).toBe(false);
  });

  it('requests the next page with the stored cursor and appends it', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a', 'b'), nextCursor: 'after-b' })
      .mockResolvedValueOnce({ items: rows('c'), nextCursor: undefined });
    const { controller } = createController(fetchPage);

    await controller.reload();
    await controller.loadNextPage();

    expect(fetchPage).toHaveBeenLastCalledWith('after-b');
    expect(getLoadedIds(controller)).toEqual(['a', 'b', 'c']);
    expect(controller.getState().list.nextCursor).toBeUndefined();
  });

  it('requests nothing more once the last page is loaded', async () => {
    const fetchPage = vi.fn().mockResolvedValue({ items: rows('a'), nextCursor: undefined });
    const { controller } = createController(fetchPage);

    await controller.reload();
    await controller.loadNextPage();

    expect(fetchPage).toHaveBeenCalledTimes(1);
  });

  it('refreshes with the first page only and keeps the scrolled tail', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a', 'b'), nextCursor: 'after-b' })
      .mockResolvedValueOnce({ items: rows('c', 'd'), nextCursor: 'after-d' })
      .mockResolvedValueOnce({ items: rows('new', 'a'), nextCursor: 'after-a' });
    const { controller } = createController(fetchPage);

    await controller.reload();
    await controller.loadNextPage();
    await controller.refreshFirstPage();

    expect(fetchPage).toHaveBeenLastCalledWith(undefined);
    expect(getLoadedIds(controller)).toEqual(['new', 'a', 'b', 'c', 'd']);
    expect(controller.getState().list.nextCursor).toBe('after-d');
  });

  it('puts a row that is not loaded at the top and keeps the cursor', async () => {
    const fetchPage = vi.fn().mockResolvedValue({ items: rows('a', 'b'), nextCursor: 'after-b' });
    const { controller } = createController(fetchPage);

    await controller.reload();
    controller.replaceOrPrependItem({ id: 'started' });

    expect(getLoadedIds(controller)).toEqual(['started', 'a', 'b']);
    expect(controller.getState().list.nextCursor).toBe('after-b');
  });

  it('replaces the loaded copy in its place when the row is already loaded', async () => {
    const fetchPage = vi.fn().mockResolvedValue({
      items: [
        { id: 'a', status: 'running' },
        { id: 'b', status: 'running' },
      ],
    });
    const { controller } = createController(fetchPage);

    await controller.reload();
    controller.replaceOrPrependItem({ id: 'b', status: 'canceled' });

    expect(controller.getState().list.items).toEqual([
      { id: 'a', status: 'running' },
      { id: 'b', status: 'canceled' },
    ]);
  });

  it('skips a refresh while the previous one is in flight', async () => {
    const pendingRefresh = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a'), nextCursor: undefined })
      .mockReturnValueOnce(pendingRefresh.promise);
    const { controller } = createController(fetchPage);

    await controller.reload();
    const firstRefresh = controller.refreshFirstPage();
    await controller.refreshFirstPage();

    pendingRefresh.resolvePage({ items: rows('new', 'a'), nextCursor: undefined });
    await firstRefresh;

    expect(fetchPage).toHaveBeenCalledTimes(2);
    expect(getLoadedIds(controller)).toEqual(['new', 'a']);
  });

  it('skips a next-page request while the previous one is in flight', async () => {
    const pendingNextPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a'), nextCursor: 'after-a' })
      .mockReturnValueOnce(pendingNextPage.promise);
    const { controller } = createController(fetchPage);

    await controller.reload();
    const firstLoad = controller.loadNextPage();
    await controller.loadNextPage();

    pendingNextPage.resolvePage({ items: rows('b'), nextCursor: undefined });
    await firstLoad;

    expect(fetchPage).toHaveBeenCalledTimes(2);
    expect(getLoadedIds(controller)).toEqual(['a', 'b']);
  });

  it('drops a refresh that answers after a reload', async () => {
    const staleRefresh = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('old'), nextCursor: undefined })
      .mockReturnValueOnce(staleRefresh.promise)
      .mockResolvedValueOnce({ items: rows('filtered'), nextCursor: undefined });
    const { controller } = createController(fetchPage);

    await controller.reload();
    const refresh = controller.refreshFirstPage();
    await controller.reload();

    staleRefresh.resolvePage({ items: rows('old', 'stale'), nextCursor: undefined });
    await refresh;

    expect(getLoadedIds(controller)).toEqual(['filtered']);
  });

  it('drops a next page that answers after a reload', async () => {
    const staleNextPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('old'), nextCursor: 'after-old' })
      .mockReturnValueOnce(staleNextPage.promise)
      .mockResolvedValueOnce({ items: rows('filtered'), nextCursor: undefined });
    const { controller } = createController(fetchPage);

    await controller.reload();
    const nextPageLoad = controller.loadNextPage();
    await controller.reload();

    staleNextPage.resolvePage({ items: rows('stale'), nextCursor: undefined });
    await nextPageLoad;

    expect(getLoadedIds(controller)).toEqual(['filtered']);
    expect(controller.getState().isLoadingMore).toBe(false);
  });

  it('reports an error from the current list and ignores one from a replaced list', async () => {
    const staleRefresh = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a'), nextCursor: undefined })
      .mockReturnValueOnce(staleRefresh.promise.then(() => Promise.reject(new Error('stale'))))
      .mockResolvedValueOnce({ items: rows('b'), nextCursor: undefined })
      .mockRejectedValueOnce(new Error('current'));
    const { controller, onError } = createController(fetchPage);

    await controller.reload();
    const refresh = controller.refreshFirstPage();
    await controller.reload();

    staleRefresh.resolvePage({ items: [], nextCursor: undefined });
    await refresh;
    await controller.refreshFirstPage();

    expect(onError).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith(new Error('current'));
  });
});
