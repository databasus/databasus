import { describe, expect, it } from 'vitest';

import { collectCursorPages } from './collectCursorPages';

describe('collectCursorPages', () => {
  it('preserves order across pages including an empty last page', async () => {
    const cursors: Array<string | undefined> = [];
    const items = await collectCursorPages<number, string>(async (cursor) => {
      cursors.push(cursor);
      if (cursor === undefined) return { items: [3, 2], nextCursor: 'second' };
      if (cursor === 'second') return { items: [1], nextCursor: 'last' };
      return { items: [] };
    });
    expect(items).toEqual([3, 2, 1]);
    expect(cursors).toEqual([undefined, 'second', 'last']);
  });

  it('returns an empty list when the first page has no items or cursor', async () => {
    let requestCount = 0;
    expect(
      await collectCursorPages(async () => {
        requestCount++;
        return { items: [] };
      }),
    ).toEqual([]);
    expect(requestCount).toBe(1);
  });

  it('propagates a later-page failure without retrying or returning a partial list', async () => {
    const unavailablePage = new Error('Page unavailable');
    const cursors: Array<string | undefined> = [];
    await expect(
      collectCursorPages<number, string>(async (cursor) => {
        cursors.push(cursor);
        if (cursor) throw unavailablePage;
        return { items: [1], nextCursor: 'second' };
      }),
    ).rejects.toBe(unavailablePage);
    expect(cursors).toEqual([undefined, 'second']);
  });
});
