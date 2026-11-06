import { describe, expect, it, vi } from 'vitest';

import type { PagedListPage } from '@/shared/lib/pagedList';
import { PagedListController } from '@/shared/lib/PagedListController';

interface Row {
  id: string;
  status?: string;
}

type RowPage = PagedListPage<Row, string>;

const rows = (...ids: string[]): Row[] => ids.map((id) => ({ id }));

const createDeferredPage = () => {
  let resolvePage: (page: RowPage) => void = () => {};
  let rejectPage: (error: Error) => void = () => {};
  const promise = new Promise<RowPage>((resolve, reject) => {
    resolvePage = resolve;
    rejectPage = reject;
  });

  return { promise, resolvePage, rejectPage };
};

const createController = (fetchPage: (cursor: string | undefined) => Promise<RowPage>) => {
  const onError = vi.fn();
  const onStateChange = vi.fn();
  const controller = new PagedListController<Row, string>(
    { fetchPage, getItemKey: (row) => row.id, onError },
    onStateChange,
  );

  return { controller, onError, onStateChange };
};

const getLoadedIds = (controller: PagedListController<Row, string>) =>
  controller.getState().list.items.map((row) => row.id);

describe('PagedListController', () => {
  it.each([
    {
      name: 'a paged first page',
      loadedPages: [{ items: rows('a', 'b'), nextCursor: 'after-b' }],
      refreshedPage: { items: rows('a', 'b'), nextCursor: 'after-b' },
    },
    {
      name: 'a complete first page',
      loadedPages: [{ items: rows('a', 'b') }],
      refreshedPage: { items: rows('a', 'b') },
    },
    {
      name: 'an empty complete list',
      loadedPages: [{ items: [] }],
      refreshedPage: { items: [] },
    },
    {
      name: 'a preserved loaded tail',
      loadedPages: [
        { items: rows('a', 'b'), nextCursor: 'after-b' },
        { items: rows('c', 'd'), nextCursor: 'after-d' },
      ],
      refreshedPage: { items: rows('a', 'b'), nextCursor: 'after-b' },
    },
  ])(
    'keeps the list reference when the refreshed page is unchanged with $name',
    async ({ loadedPages, refreshedPage }) => {
      const fetchPage = vi.fn();
      for (const page of loadedPages) fetchPage.mockResolvedValueOnce(page);
      fetchPage.mockResolvedValueOnce(refreshedPage);
      const { controller, onStateChange } = createController(fetchPage);

      await controller.reload();
      for (let pageNumber = 1; pageNumber < loadedPages.length; pageNumber += 1) {
        await controller.loadNextPage();
      }
      const loadedState = controller.getState();
      onStateChange.mockClear();

      await controller.refreshFirstPage();

      expect(controller.getState().list).toBe(loadedState.list);
      expect(controller.getState()).toBe(loadedState);
      expect(onStateChange).not.toHaveBeenCalled();
    },
  );

  it.each([
    {
      name: 'a row changes status',
      firstPage: { items: [{ id: 'a', status: 'running' }] },
      refreshedPage: { items: [{ id: 'a', status: 'completed' }] },
    },
    {
      name: 'rows change order',
      firstPage: { items: rows('a', 'b') },
      refreshedPage: { items: rows('b', 'a') },
    },
    {
      name: 'a nested value changes',
      firstPage: { items: [{ id: 'a', storage: { name: 'Local', isAvailable: false } }] },
      refreshedPage: { items: [{ id: 'a', storage: { name: 'Local', isAvailable: true } }] },
    },
    {
      name: 'a nested array changes',
      firstPage: { items: [{ id: 'a', logs: ['Starting'] }] },
      refreshedPage: { items: [{ id: 'a', logs: ['Starting', 'Finished'] }] },
    },
    {
      name: 'a null value is replaced',
      firstPage: { items: [{ id: 'a', error: null }] },
      refreshedPage: { items: [{ id: 'a', error: { message: 'Failed' } }] },
    },
    {
      name: 'the effective continuation cursor changes',
      firstPage: { items: rows('a', 'b'), nextCursor: 'first-cursor' },
      refreshedPage: { items: rows('a', 'b'), nextCursor: 'new-cursor' },
    },
  ])('publishes the refreshed list when $name', async ({ firstPage, refreshedPage }) => {
    const fetchPage = vi.fn().mockResolvedValueOnce(firstPage).mockResolvedValueOnce(refreshedPage);
    const { controller, onStateChange } = createController(fetchPage);

    await controller.reload();
    const loadedState = controller.getState();
    onStateChange.mockClear();

    await controller.refreshFirstPage();

    expect(controller.getState().list).toEqual(refreshedPage);
    expect(controller.getState()).not.toBe(loadedState);
    expect(onStateChange).toHaveBeenCalledOnce();
  });

  it('keeps the list reference when object property order changes', async () => {
    const firstPage = {
      items: [{ id: 'a', status: 'completed', storage: { name: 'Local', errors: [] } }],
    };
    const refreshedPage = {
      items: [{ storage: { errors: [], name: 'Local' }, status: 'completed', id: 'a' }],
      nextCursor: undefined,
      metadata: undefined,
    };
    const fetchPage = vi.fn().mockResolvedValueOnce(firstPage).mockResolvedValueOnce(refreshedPage);
    const { controller, onStateChange } = createController(fetchPage);

    await controller.reload();
    const loadedState = controller.getState();
    onStateChange.mockClear();

    await controller.refreshFirstPage();

    expect(controller.getState()).toBe(loadedState);
    expect(onStateChange).not.toHaveBeenCalled();
  });

  it.each([
    {
      name: 'retains list state when total usage is unchanged',
      refreshedUsageMb: 100,
      shouldPublish: false,
    },
    {
      name: 'publishes the list when only total usage changes',
      refreshedUsageMb: 200,
      shouldPublish: true,
    },
  ])('$name', async ({ refreshedUsageMb, shouldPublish }) => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a'), metadata: 100 })
      .mockResolvedValueOnce({ items: rows('a'), metadata: refreshedUsageMb });
    const onStateChange = vi.fn();
    const controller = new PagedListController<Row, string, number>(
      { fetchPage, getItemKey: (row) => row.id, onError: vi.fn() },
      onStateChange,
    );

    await controller.reload();
    const loadedState = controller.getState();
    onStateChange.mockClear();

    await controller.refreshFirstPage();

    expect(controller.getState().list.metadata).toBe(refreshedUsageMb);
    expect(onStateChange).toHaveBeenCalledTimes(shouldPublish ? 1 : 0);

    if (!shouldPublish) expect(controller.getState()).toBe(loadedState);
  });

  it('preserves the next page and loading state when an unchanged refresh overlaps it', async () => {
    const nextPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a', 'b'), nextCursor: 'after-b' })
      .mockReturnValueOnce(nextPage.promise)
      .mockResolvedValueOnce({ items: rows('a', 'b'), nextCursor: 'after-b' });
    const { controller, onStateChange } = createController(fetchPage);

    await controller.reload();
    const pendingNext = controller.loadNextPage();
    const loadingState = controller.getState();
    onStateChange.mockClear();

    await controller.refreshFirstPage();

    expect(controller.getState()).toBe(loadingState);
    expect(controller.getState().isLoadingMore).toBe(true);
    expect(onStateChange).not.toHaveBeenCalled();

    await controller.loadNextPage();

    expect(fetchPage).toHaveBeenCalledTimes(3);

    nextPage.resolvePage({ items: rows('c') });
    await pendingNext;

    expect(getLoadedIds(controller)).toEqual(['a', 'b', 'c']);
    expect(controller.getState().isLoadingMore).toBe(false);
  });

  it.each([
    { name: 'the same rows become complete', refreshedPage: { items: rows('a', 'b') } },
    { name: 'the list becomes empty', refreshedPage: { items: [] } },
    {
      name: 'the effective cursor changes',
      refreshedPage: { items: rows('a', 'b'), nextCursor: 'new-cursor' },
    },
  ])('invalidates a pending next page when $name', async ({ refreshedPage }) => {
    const nextPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a', 'b'), nextCursor: 'after-b' })
      .mockReturnValueOnce(nextPage.promise)
      .mockResolvedValueOnce(refreshedPage);
    const { controller, onStateChange } = createController(fetchPage);

    await controller.reload();
    const pendingNext = controller.loadNextPage();
    onStateChange.mockClear();

    await controller.refreshFirstPage();

    expect(controller.getState().list).toEqual(refreshedPage);
    expect(controller.getState().isLoadingMore).toBe(false);
    expect(onStateChange).toHaveBeenCalledOnce();

    const refreshedState = controller.getState();

    nextPage.resolvePage({ items: rows('stale') });
    await pendingNext;

    expect(controller.getState()).toBe(refreshedState);
    expect(onStateChange).toHaveBeenCalledOnce();
  });

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

  it('stays silent when a refresh fails for the current list or a replaced list', async () => {
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

    expect(onError).not.toHaveBeenCalled();
    expect(controller.getState().firstLoadError).toBeUndefined();
    expect(getLoadedIds(controller)).toEqual(['b']);
  });

  it('blocks scroll and polling until the replacement first page succeeds', async () => {
    const firstPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('old'), nextCursor: 'old-cursor' })
      .mockReturnValueOnce(firstPage.promise);
    const { controller } = createController(fetchPage);

    await controller.reload();
    const reload = controller.reload();
    await controller.loadNextPage();
    await controller.refreshFirstPage();

    expect(fetchPage).toHaveBeenCalledTimes(2);
    expect(controller.getState().list).toEqual({ items: [] });

    firstPage.resolvePage({ items: rows('new'), nextCursor: 'new-cursor' });
    await reload;
    expect(getLoadedIds(controller)).toEqual(['new']);
  });

  it('keeps a new next-page loading flag when an old request finishes', async () => {
    const oldPage = createDeferredPage();
    const newPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('old'), nextCursor: 'old-cursor' })
      .mockReturnValueOnce(oldPage.promise)
      .mockResolvedValueOnce({ items: rows('new'), nextCursor: 'new-cursor' })
      .mockReturnValueOnce(newPage.promise);
    const { controller } = createController(fetchPage);

    await controller.reload();
    const oldNext = controller.loadNextPage();
    await controller.reload();
    const newNext = controller.loadNextPage();

    oldPage.resolvePage({ items: rows('stale') });
    await oldNext;
    expect(controller.getState().isLoadingMore).toBe(true);

    await controller.loadNextPage();
    expect(fetchPage).toHaveBeenCalledTimes(4);

    newPage.resolvePage({ items: rows('current') });
    await newNext;
    expect(getLoadedIds(controller)).toEqual(['new', 'current']);
  });

  it('allows a new refresh while an old generation is pending and keeps its own lock', async () => {
    const oldPage = createDeferredPage();
    const newPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('old') })
      .mockReturnValueOnce(oldPage.promise)
      .mockResolvedValueOnce({ items: rows('new') })
      .mockReturnValueOnce(newPage.promise);
    const { controller } = createController(fetchPage);

    await controller.reload();
    const oldRefresh = controller.refreshFirstPage();
    await controller.reload();
    const newRefresh = controller.refreshFirstPage();

    expect(fetchPage).toHaveBeenCalledTimes(4);
    oldPage.resolvePage({ items: rows('stale') });
    await oldRefresh;
    await controller.refreshFirstPage();
    expect(fetchPage).toHaveBeenCalledTimes(4);

    newPage.resolvePage({ items: rows('current') });
    await newRefresh;
    expect(getLoadedIds(controller)).toEqual(['current']);
  });

  it('requires an explicit reload after the first page fails', async () => {
    const fetchPage = vi
      .fn()
      .mockRejectedValueOnce(new Error('unavailable'))
      .mockResolvedValueOnce({ items: rows('recovered') });
    const { controller, onError } = createController(fetchPage);

    await controller.reload();
    expect(controller.getState().isLoading).toBe(false);
    await controller.refreshFirstPage();
    await controller.loadNextPage();
    expect(fetchPage).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledOnce();

    await controller.reload();
    expect(getLoadedIds(controller)).toEqual(['recovered']);
  });

  it('ignores errors from an obsolete first page and next page', async () => {
    const oldPage = createDeferredPage();
    const oldNextPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockReturnValueOnce(oldPage.promise)
      .mockResolvedValueOnce({ items: rows('new'), nextCursor: 'next' })
      .mockReturnValueOnce(oldNextPage.promise)
      .mockResolvedValueOnce({ items: rows('current') });
    const { controller, onError } = createController(fetchPage);

    const oldReload = controller.reload();
    await controller.reload();
    const oldNext = controller.loadNextPage();
    await controller.reload();

    oldPage.rejectPage(new Error('old first'));
    oldNextPage.rejectPage(new Error('old next'));
    await Promise.all([oldReload, oldNext]);

    expect(onError).not.toHaveBeenCalled();
    expect(getLoadedIds(controller)).toEqual(['current']);
  });

  it.each(['refresh', 'next'])(
    'preserves aligned rows and cursor when %s finishes first',
    async (first) => {
      const refreshPage = createDeferredPage();
      const nextPage = createDeferredPage();
      const fetchPage = vi
        .fn()
        .mockResolvedValueOnce({ items: rows('a', 'b', 'c'), nextCursor: 'after-c' })
        .mockReturnValueOnce(refreshPage.promise)
        .mockReturnValueOnce(nextPage.promise);
      const { controller } = createController(fetchPage);

      await controller.reload();
      const refresh = controller.refreshFirstPage();
      const next = controller.loadNextPage();

      const finishRefresh = async () => {
        refreshPage.resolvePage({
          items: [{ id: 'new' }, { id: 'a', status: 'completed' }],
          nextCursor: 'after-a',
        });
        await refresh;
      };
      const finishNext = async () => {
        nextPage.resolvePage({
          items: [{ id: 'a', status: 'pending' }, ...rows('c', 'd')],
          nextCursor: 'after-d',
        });
        await next;
      };

      if (first === 'refresh') {
        await finishRefresh();
        await finishNext();
      } else {
        await finishNext();
        await finishRefresh();
      }

      expect(getLoadedIds(controller)).toEqual(['new', 'a', 'b', 'c', 'd']);
      expect(controller.getState().list.items.find((row) => row.id === 'a')?.status).toBe(
        'completed',
      );
      expect(controller.getState().list.nextCursor).toBe('after-d');
    },
  );

  it.each(['refresh', 'next'])(
    'keeps compatible pagination after an unchanged extent refresh when %s finishes first',
    async (first) => {
      const nextPage = createDeferredPage();
      const refreshPage = createDeferredPage();
      const fetchPage = vi
        .fn()
        .mockResolvedValueOnce({ items: rows('a', 'b'), nextCursor: 'after-b' })
        .mockReturnValueOnce(nextPage.promise)
        .mockReturnValueOnce(refreshPage.promise);
      const { controller } = createController(fetchPage);

      await controller.reload();
      const next = controller.loadNextPage();
      const refresh = controller.refreshFirstPage();

      const finishRefresh = async () => {
        refreshPage.resolvePage({
          items: [{ id: 'a' }, { id: 'b', status: 'completed' }],
          nextCursor: 'after-b',
        });
        await refresh;
      };
      const finishNext = async () => {
        nextPage.resolvePage({ items: rows('c', 'd'), nextCursor: 'after-d' });
        await next;
      };

      if (first === 'refresh') {
        await finishRefresh();
        expect(controller.getState().isLoadingMore).toBe(true);
        await controller.loadNextPage();
        expect(fetchPage).toHaveBeenCalledTimes(3);
        await finishNext();
      } else {
        await finishNext();
        await finishRefresh();
      }

      expect(getLoadedIds(controller)).toEqual(['a', 'b', 'c', 'd']);
      expect(controller.getState().list.nextCursor).toBe('after-d');
      expect(controller.getState().list.items.find((row) => row.id === 'b')?.status).toBe(
        'completed',
      );
      expect(controller.getState().isLoadingMore).toBe(false);
    },
  );

  it('makes older surviving rows reachable through the new cursor after overlap disappears', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('old-a', 'old-b'), nextCursor: 'old-cursor' })
      .mockResolvedValueOnce({ items: rows('new-a', 'new-b'), nextCursor: 'new-cursor' })
      .mockResolvedValueOnce({ items: rows('old-a', 'old-b') });
    const { controller } = createController(fetchPage);

    await controller.reload();
    await controller.refreshFirstPage();
    expect(getLoadedIds(controller)).toEqual(['new-a', 'new-b']);

    await controller.loadNextPage();
    expect(fetchPage).toHaveBeenLastCalledWith('new-cursor');
    expect(getLoadedIds(controller)).toEqual(['new-a', 'new-b', 'old-a', 'old-b']);
  });

  it.each(['refresh', 'next'])(
    'keeps a complete refreshed page authoritative when %s finishes first',
    async (first) => {
      const refreshPage = createDeferredPage();
      const nextPage = createDeferredPage();
      const fetchPage = vi
        .fn()
        .mockResolvedValueOnce({ items: rows('a', 'b'), nextCursor: 'after-b' })
        .mockReturnValueOnce(refreshPage.promise)
        .mockReturnValueOnce(nextPage.promise);
      const { controller } = createController(fetchPage);

      await controller.reload();
      const refresh = controller.refreshFirstPage();
      const next = controller.loadNextPage();

      const finishRefresh = async () => {
        refreshPage.resolvePage({ items: rows('a') });
        await refresh;
      };
      const finishNext = async () => {
        nextPage.resolvePage({ items: rows('c') });
        await next;
      };

      if (first === 'refresh') {
        await finishRefresh();
        await finishNext();
      } else {
        await finishNext();
        await finishRefresh();
      }

      expect(getLoadedIds(controller)).toEqual(['a']);
      expect(controller.getState().list.nextCursor).toBeUndefined();
    },
  );

  it.each(['success', 'error'])(
    'publishes no state or errors when a pending refresh returns %s after deactivation',
    async (outcome) => {
      const refreshPage = createDeferredPage();
      const fetchPage = vi
        .fn()
        .mockResolvedValueOnce({ items: rows('current'), nextCursor: 'after-current' })
        .mockReturnValueOnce(refreshPage.promise);
      const { controller, onError, onStateChange } = createController(fetchPage);

      await controller.reload();

      const refresh = controller.refreshFirstPage();
      controller.deactivate();
      const deactivatedState = controller.getState();
      onStateChange.mockClear();

      if (outcome === 'success') refreshPage.resolvePage({ items: rows('late') });
      else refreshPage.rejectPage(new Error('late refresh'));

      await refresh;

      expect(controller.getState()).toBe(deactivatedState);
      expect(onStateChange).not.toHaveBeenCalled();
      expect(onError).not.toHaveBeenCalled();
    },
  );

  it.each(['success', 'error'])(
    'publishes no state or errors after deactivation with a pending %s',
    async (outcome) => {
      const page = createDeferredPage();
      const onError = vi.fn();
      const onStateChange = vi.fn();
      const controller = new PagedListController<Row, string>(
        { fetchPage: () => page.promise, getItemKey: (row) => row.id, onError },
        onStateChange,
      );

      const reload = controller.reload();
      controller.deactivate();
      const publications = onStateChange.mock.calls.length;

      if (outcome === 'success') page.resolvePage({ items: rows('late') });
      else page.rejectPage(new Error('late error'));
      await reload;

      expect(onStateChange).toHaveBeenCalledTimes(publications);
      expect(onError).not.toHaveBeenCalled();
    },
  );

  it('stays silent with its rows, cursor and pending next page when a refresh fails', async () => {
    const nextPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a'), nextCursor: 'after-a' })
      .mockReturnValueOnce(nextPage.promise)
      .mockRejectedValueOnce(new Error('unavailable'))
      .mockResolvedValueOnce({ items: rows('fresh'), nextCursor: 'after-a' });
    const { controller, onError } = createController(fetchPage);

    await controller.reload();
    const pendingNext = controller.loadNextPage();
    await controller.refreshFirstPage();

    expect(onError).not.toHaveBeenCalled();
    expect(getLoadedIds(controller)).toEqual(['a']);
    expect(controller.getState().list.nextCursor).toBe('after-a');
    expect(controller.getState().isLoadingMore).toBe(true);
    expect(controller.getState().firstLoadError).toBeUndefined();

    await controller.refreshFirstPage();

    expect(fetchPage).toHaveBeenCalledTimes(4);
    expect(getLoadedIds(controller)).toEqual(['fresh']);

    nextPage.resolvePage({ items: rows('b') });
    await pendingNext;

    expect(getLoadedIds(controller)).toEqual(['fresh', 'b']);
    expect(controller.getState().isLoadingMore).toBe(false);
  });

  it('keeps a failed first page until reload clears it', async () => {
    const failure = new Error('unavailable');
    const fetchPage = vi
      .fn()
      .mockRejectedValueOnce(failure)
      .mockResolvedValueOnce({ items: rows('recovered') });
    const { controller, onError } = createController(fetchPage);

    await controller.reload();

    expect(controller.getState().firstLoadError).toBe(failure);
    expect(onError).toHaveBeenCalledOnce();
    expect(onError).toHaveBeenCalledWith(failure, 'first');

    const retry = controller.reload();

    expect(controller.getState().firstLoadError).toBeUndefined();
    expect(controller.getState().isLoading).toBe(true);

    await retry;

    expect(controller.getState().firstLoadError).toBeUndefined();
    expect(getLoadedIds(controller)).toEqual(['recovered']);
  });

  it('publishes nothing when an obsolete first page fails', async () => {
    const oldPage = createDeferredPage();
    const fetchPage = vi
      .fn()
      .mockReturnValueOnce(oldPage.promise)
      .mockResolvedValueOnce({ items: rows('new') });
    const { controller, onError } = createController(fetchPage);

    const oldReload = controller.reload();
    await controller.reload();
    oldPage.rejectPage(new Error('old first'));
    await oldReload;

    expect(controller.getState().firstLoadError).toBeUndefined();
    expect(onError).not.toHaveBeenCalled();
  });

  it('rejects a current next page once with its rows and cursor before requesting it again', async () => {
    const failure = new Error('unavailable');
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: rows('a'), nextCursor: 'after-a' })
      .mockRejectedValueOnce(failure)
      .mockResolvedValueOnce({ items: rows('b') });
    const { controller, onError } = createController(fetchPage);

    await controller.reload();
    await controller.loadNextPage();

    expect(onError).toHaveBeenCalledOnce();
    expect(onError).toHaveBeenCalledWith(failure, 'next');
    expect(getLoadedIds(controller)).toEqual(['a']);
    expect(controller.getState().list.nextCursor).toBe('after-a');
    expect(controller.getState().firstLoadError).toBeUndefined();

    await controller.loadNextPage();

    expect(fetchPage).toHaveBeenLastCalledWith('after-a');
    expect(getLoadedIds(controller)).toEqual(['a', 'b']);
  });
});
