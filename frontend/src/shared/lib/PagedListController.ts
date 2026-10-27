import { type PagedListPage, appendNextPage, mergeRefreshedFirstPage } from './pagedList';

export interface PagedListOptions<TItem, TCursor> {
  fetchPage: (cursor: TCursor | undefined) => Promise<PagedListPage<TItem, TCursor>>;
  getItemKey: (item: TItem) => string;
  onError: (error: unknown) => void;
}

export interface PagedListState<TItem, TCursor> {
  list: PagedListPage<TItem, TCursor>;
  isLoading: boolean;
  isLoadingMore: boolean;
}

export const createInitialPagedListState = <TItem, TCursor>(): PagedListState<TItem, TCursor> => ({
  list: { items: [] },
  isLoading: true,
  isLoadingMore: false,
});

// A response that arrives after a reload belongs to the previous list and is
// dropped. The class holds no React state, so the paging rules are testable
// without rendering.
export class PagedListController<TItem, TCursor> {
  private state = createInitialPagedListState<TItem, TCursor>();

  private generation = 0;
  private isRefreshInFlight = false;

  constructor(
    private options: PagedListOptions<TItem, TCursor>,
    private readonly onStateChange: (state: PagedListState<TItem, TCursor>) => void,
  ) {}

  setOptions(options: PagedListOptions<TItem, TCursor>) {
    this.options = options;
  }

  getState(): PagedListState<TItem, TCursor> {
    return this.state;
  }

  async reload() {
    this.generation += 1;
    const generation = this.generation;

    this.updateState({ isLoading: true });

    try {
      const firstPage = await this.options.fetchPage(undefined);
      if (generation !== this.generation) return;

      this.updateState({ list: firstPage, isLoading: false });
    } catch (e) {
      if (generation !== this.generation) return;

      this.updateState({ isLoading: false });
      this.options.onError(e);
    }
  }

  async refreshFirstPage() {
    if (this.isRefreshInFlight) return;
    this.isRefreshInFlight = true;

    const generation = this.generation;

    try {
      const firstPage = await this.options.fetchPage(undefined);
      if (generation !== this.generation) return;

      this.updateState({
        list: mergeRefreshedFirstPage(this.state.list, firstPage, this.options.getItemKey),
      });
    } catch (e) {
      if (generation === this.generation) this.options.onError(e);
    } finally {
      this.isRefreshInFlight = false;
    }
  }

  // Shows a row a request answered with before the next refresh lists it: the loaded copy is
  // replaced, and a new row goes to the top of the newest-first list.
  replaceOrPrependItem(item: TItem) {
    const itemKey = this.options.getItemKey(item);
    const loadedItems = this.state.list.items;
    const isLoaded = loadedItems.some(
      (loadedItem) => this.options.getItemKey(loadedItem) === itemKey,
    );

    const items = isLoaded
      ? loadedItems.map((loadedItem) =>
          this.options.getItemKey(loadedItem) === itemKey ? item : loadedItem,
        )
      : [item, ...loadedItems];

    this.updateState({ list: { ...this.state.list, items } });
  }

  async loadNextPage() {
    const cursor = this.state.list.nextCursor;
    if (this.state.isLoadingMore || cursor === undefined) return;

    const generation = this.generation;

    this.updateState({ isLoadingMore: true });

    try {
      const nextPage = await this.options.fetchPage(cursor);
      if (generation !== this.generation) return;

      this.updateState({
        list: appendNextPage(this.state.list, nextPage, this.options.getItemKey),
      });
    } catch (e) {
      if (generation === this.generation) this.options.onError(e);
    } finally {
      this.updateState({ isLoadingMore: false });
    }
  }

  private updateState(change: Partial<PagedListState<TItem, TCursor>>) {
    this.state = { ...this.state, ...change };
    this.onStateChange(this.state);
  }
}
