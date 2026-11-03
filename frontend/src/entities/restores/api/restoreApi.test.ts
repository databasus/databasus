import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { restoreApi } from './restoreApi';

const mockServer = setUpMockApi();

describe('restoreApi.listAllRestores', () => {
  it('returns the restores of every page and sends the backup, limit and cursor', async () => {
    const requestedQueries: URLSearchParams[] = [];
    mockServer.use(
      http.get('*/api/v1/restores', ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        return HttpResponse.json(
          query.get('after') === null
            ? { items: [{ id: 'first-restore' }], nextCursor: 'second-page' }
            : { items: [{ id: 'second-restore' }] },
        );
      }),
    );

    const listedRestores = await restoreApi.listAllRestores('backup-id');

    expect(listedRestores.map((restore) => restore.id)).toEqual([
      'first-restore',
      'second-restore',
    ]);
    expect(requestedQueries.map((query) => query.get('after'))).toEqual([null, 'second-page']);
    expect(requestedQueries.map((query) => query.get('backupId'))).toEqual([
      'backup-id',
      'backup-id',
    ]);
    expect(requestedQueries.map((query) => query.get('limit'))).toEqual(['100', '100']);
  });
});
