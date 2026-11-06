import { describe, expect, it } from 'vitest';

import {
  type PagedListPage,
  appendNextPage,
  mergeRefreshedFirstPage,
} from '@/shared/lib/pagedList';

interface Row {
  id: string;
  label: string;
}

const getRowKey = (row: Row) => row.id;

const rows = (...ids: string[]): Row[] => ids.map((id) => ({ id, label: id }));

const ids = (list: PagedListPage<Row, string>) => list.items.map(getRowKey);

const expectNoDuplicateKeys = (list: PagedListPage<Row, string>) => {
  expect(new Set(ids(list)).size).toBe(list.items.length);
};

describe('appendNextPage', () => {
  it('appends the page and takes its cursor', () => {
    const merged = appendNextPage(
      { items: rows('a', 'b'), nextCursor: 'after-b' },
      { items: rows('c', 'd'), nextCursor: 'after-d' },
      getRowKey,
    );

    expect(ids(merged)).toEqual(['a', 'b', 'c', 'd']);
    expect(merged.nextCursor).toBe('after-d');
  });

  it('drops rows the list already holds', () => {
    const merged = appendNextPage(
      { items: rows('new', 'a', 'b'), nextCursor: '2' },
      { items: rows('b', 'c'), nextCursor: undefined },
      getRowKey,
    );

    expect(ids(merged)).toEqual(['new', 'a', 'b', 'c']);
    expect(merged.nextCursor).toBeUndefined();
    expectNoDuplicateKeys(merged);
  });
});

describe('mergeRefreshedFirstPage', () => {
  it('takes the first page when only the first page is loaded', () => {
    const firstPage = { items: [{ id: 'a', label: 'a-completed' }, ...rows('b')], nextCursor: 'b' };

    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'b'), nextCursor: 'after-b' },
      firstPage,
      getRowKey,
    );

    expect(merged).toEqual(firstPage);
  });

  it('keeps a loaded row that a new row pushed off the first page', () => {
    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'b'), nextCursor: 'after-b' },
      { items: rows('new', 'a'), nextCursor: 'after-a' },
      getRowKey,
    );

    expect(ids(merged)).toEqual(['new', 'a', 'b']);
    expect(merged.nextCursor).toBe('after-b');
  });

  it('puts new rows on top and keeps the tail loaded by scrolling', () => {
    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'b', 'c', 'd', 'e'), nextCursor: 'after-e' },
      { items: rows('new', 'a'), nextCursor: 'after-a' },
      getRowKey,
    );

    expect(ids(merged)).toEqual(['new', 'a', 'b', 'c', 'd', 'e']);
    expect(merged.nextCursor).toBe('after-e');
    expectNoDuplicateKeys(merged);
  });

  it('replaces a changed row in place', () => {
    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'b', 'c'), nextCursor: 'after-c' },
      { items: [{ id: 'a', label: 'a-completed' }, ...rows('b')], nextCursor: 'after-b' },
      getRowKey,
    );

    expect(merged.items).toEqual([{ id: 'a', label: 'a-completed' }, ...rows('b', 'c')]);
  });

  it('drops loaded rows that left the first page', () => {
    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'deleted', 'b', 'c'), nextCursor: 'after-c' },
      { items: rows('a', 'b'), nextCursor: 'after-b' },
      getRowKey,
    );

    expect(ids(merged)).toEqual(['a', 'b', 'c']);
  });

  it('falls back to the first page when its last row is not loaded', () => {
    const firstPage = { items: rows('x', 'y'), nextCursor: 'after-y' };

    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'b', 'c'), nextCursor: 'after-c' },
      firstPage,
      getRowKey,
    );

    expect(merged).toEqual(firstPage);
  });

  it('keeps one row when two ranges merge under the older range id', () => {
    const merged = mergeRefreshedFirstPage(
      { items: rows('full', 'newer-range', 'older-range', 'incr', 'oldest'), nextCursor: 'c' },
      { items: rows('full', 'older-range'), nextCursor: 'after-older-range' },
      getRowKey,
    );

    expect(ids(merged)).toEqual(['full', 'older-range', 'incr', 'oldest']);
    expectNoDuplicateKeys(merged);
  });

  it('keeps no rows when the refreshed first page is empty', () => {
    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'b'), nextCursor: 'after-b' },
      { items: [], nextCursor: undefined },
      getRowKey,
    );

    expect(merged.items).toEqual([]);
    expect(merged.nextCursor).toBeUndefined();
  });

  it('takes the first page when it is the whole list', () => {
    const merged = mergeRefreshedFirstPage(
      { items: rows('a', 'b', 'c'), nextCursor: 'after-c' },
      { items: rows('a', 'b'), nextCursor: undefined },
      getRowKey,
    );

    expect(ids(merged)).toEqual(['a', 'b']);
    expect(merged.nextCursor).toBeUndefined();
  });
});
