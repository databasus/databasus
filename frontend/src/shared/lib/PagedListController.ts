import {
  type PagedListPage,
  appendNextPage,
  mergeRefreshedFirstPage,
} from '@/shared/lib/pagedList';

type PagedListFailedPage = 'first' | 'next';

export interface PagedListOptions<TItem, TCursor, TMetadata = undefined> {
  fetchPage: (cursor: TCursor | undefined) => Promise<PagedListPage<TItem, TCursor, TMetadata>>;
  getItemKey: (item: TItem) => string;
  onError: (error: unknown, failedPage: PagedListFailedPage) => void;
}

export interface PagedListState<TItem, TCursor, TMetadata = undefined> {
  list: PagedListPage<TItem, TCursor, TMetadata>;
  isLoading: boolean;
  isLoadingMore: boolean;
  firstLoadError?: unknown;
}

export const createInitialPagedListState = <
  TItem,
  TCursor,
  TMetadata = undefined,
>(): PagedListState<TItem, TCursor, TMetadata> => ({
  list: { items: [] },
  isLoading: true,
  isLoadingMore: false,
  firstLoadError: undefined,
});

const isPageValueEqual = ({
  currentValue,
  refreshedValue,
}: {
  currentValue: unknown;
  refreshedValue: unknown;
}): boolean => {
  if (currentValue === refreshedValue) return true;

  if (
    currentValue === null ||
    refreshedValue === null ||
    typeof currentValue !== 'object' ||
    typeof refreshedValue !== 'object'
  ) {
    return false;
  }

  if (Array.isArray(currentValue) || Array.isArray(refreshedValue)) {
    return (
      Array.isArray(currentValue) &&
      Array.isArray(refreshedValue) &&
      currentValue.length === refreshedValue.length &&
      currentValue.every((value, index) =>
        isPageValueEqual({ currentValue: value, refreshedValue: refreshedValue[index] }),
      )
    );
  }

  const currentFields = currentValue as Record<string, unknown>;
  const refreshedFields = refreshedValue as Record<string, unknown>;
  const fieldNames = Object.keys(currentFields);

  return (
    fieldNames.length === Object.keys(refreshedFields).length &&
    fieldNames.every(
      (fieldName) =>
        Object.prototype.hasOwnProperty.call(refreshedFields, fieldName) &&
        isPageValueEqual({
          currentValue: currentFields[fieldName],
          refreshedValue: refreshedFields[fieldName],
        }),
    )
  );
};

export class PagedListController<TItem, TCursor, TMetadata = undefined> {
  private state = createInitialPagedListState<TItem, TCursor, TMetadata>();

  private generation = 0;
  private isActive = true;
  private isReady = false;
  private refreshRequest?: symbol;
  private nextPageRequest?: symbol;

  constructor(
    private options: PagedListOptions<TItem, TCursor, TMetadata>,
    private readonly onStateChange: (state: PagedListState<TItem, TCursor, TMetadata>) => void,
  ) {}

  setOptions(options: PagedListOptions<TItem, TCursor, TMetadata>) {
    this.options = options;
  }

  getState(): PagedListState<TItem, TCursor, TMetadata> {
    return this.state;
  }

  activate() {
    this.isActive = true;
  }

  deactivate() {
    this.isActive = false;
    this.invalidateRequests();
  }

  async reload() {
    if (!this.isActive) return;
    this.invalidateRequests();
    const generation = this.generation;
    this.updateState(createInitialPagedListState());
    try {
      const firstPage = await this.options.fetchPage(undefined);
      if (!this.isCurrentGeneration(generation)) return;
      this.isReady = true;
      this.updateState({ list: firstPage, isLoading: false });
    } catch (error) {
      if (!this.isCurrentGeneration(generation)) return;
      this.updateState({ isLoading: false, firstLoadError: error });
      this.options.onError(error, 'first');
    }
  }

  async refreshFirstPage() {
    if (!this.isActive || !this.isReady || this.refreshRequest) return;
    const request = Symbol();
    this.refreshRequest = request;
    const generation = this.generation;
    const isCurrent = () => this.isCurrentGeneration(generation) && this.refreshRequest === request;

    try {
      const firstPage = await this.options.fetchPage(undefined);
      if (!isCurrent()) return;

      const list = mergeRefreshedFirstPage(this.state.list, firstPage, this.options.getItemKey);
      const isPendingCursorInvalid =
        list === firstPage &&
        (firstPage.nextCursor === undefined || firstPage.nextCursor !== this.state.list.nextCursor);
      if (isPendingCursorInvalid) this.nextPageRequest = undefined;

      const isListUnchanged =
        isPageValueEqual({ currentValue: this.state.list.items, refreshedValue: list.items }) &&
        isPageValueEqual({
          currentValue: this.state.list.nextCursor,
          refreshedValue: list.nextCursor,
        }) &&
        isPageValueEqual({ currentValue: this.state.list.metadata, refreshedValue: list.metadata });
      const isLoadingMore = isPendingCursorInvalid ? false : this.state.isLoadingMore;
      if (isListUnchanged && isLoadingMore === this.state.isLoadingMore) return;

      this.updateState({ list: isListUnchanged ? this.state.list : list, isLoadingMore });
    } catch {
      // The next refresh tries again, and the shown rows stay until one succeeds.
    } finally {
      if (isCurrent()) this.refreshRequest = undefined;
    }
  }

  // Shows a row a request answered with before the next refresh lists it: the loaded copy is
  // replaced, and a new row goes to the top of the newest-first list.
  replaceOrPrependItem(item: TItem) {
    if (!this.isActive) return;
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
    if (!this.isActive || !this.isReady || this.nextPageRequest || cursor === undefined) return;
    const request = Symbol();
    this.nextPageRequest = request;
    const generation = this.generation;
    const isCurrent = () =>
      this.isCurrentGeneration(generation) && this.nextPageRequest === request;

    this.updateState({ isLoadingMore: true });
    try {
      const nextPage = await this.options.fetchPage(cursor);
      if (!isCurrent()) return;
      this.updateState({
        list: appendNextPage(this.state.list, nextPage, this.options.getItemKey),
      });
    } catch (error) {
      if (isCurrent()) this.options.onError(error, 'next');
    } finally {
      if (isCurrent()) {
        this.nextPageRequest = undefined;
        this.updateState({ isLoadingMore: false });
      }
    }
  }

  private invalidateRequests() {
    this.generation += 1;
    this.isReady = false;
    this.refreshRequest = undefined;
    this.nextPageRequest = undefined;
  }

  private isCurrentGeneration(generation: number) {
    return this.isActive && this.generation === generation;
  }

  private updateState(change: Partial<PagedListState<TItem, TCursor, TMetadata>>) {
    this.state = { ...this.state, ...change };
    this.onStateChange(this.state);
  }
}
