import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { buildProblem, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { storageApi } from './storageApi';

const mockServer = setUpMockApi();

const buildStorage = (name: string): ApiComponents['schemas']['Storage'] => ({
  id: `${name}-id`,
  workspaceId: 'workspace-id',
  name,
  settings: { type: 'LOCAL' },
});

describe('storageApi.listAllStorages', () => {
  it('returns the storages of every page when the final page is empty', async () => {
    const requestedQueries: URLSearchParams[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/storages`, ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        const storagePage: ApiComponents['schemas']['StoragePage'] =
          query.get('after') === null
            ? {
                items: [buildStorage('Alpha')],
                nextCursor: 'first-page-cursor',
              }
            : query.get('after') === 'first-page-cursor'
              ? { items: [buildStorage('Bravo')], nextCursor: 'last-page-cursor' }
              : { items: [] };

        return HttpResponse.json(storagePage);
      }),
    );

    const listedStorages = await storageApi.listAllStorages('workspace-id');

    expect(listedStorages.map((storage) => storage.name)).toEqual(['Alpha', 'Bravo']);
    expect(requestedQueries.map((query) => query.get('after'))).toEqual([
      null,
      'first-page-cursor',
      'last-page-cursor',
    ]);
    expect(requestedQueries.map((query) => query.get('workspaceId'))).toEqual([
      'workspace-id',
      'workspace-id',
      'workspace-id',
    ]);
  });
  it('rejects the complete list when a later page fails without retrying or changing filters', async () => {
    mockServer.use(
      http.get(`*${API_BASE_PATH}/storages`, ({ request }) => {
        const after = new URL(request.url).searchParams.get('after');
        return after === null
          ? HttpResponse.json({ items: [buildStorage('Alpha')], nextCursor: 'second-page' })
          : HttpResponse.json(buildProblem('internal', 500), { status: 500 });
      }),
    );

    await expect(storageApi.listAllStorages('workspace-id')).rejects.toMatchObject({
      code: 'internal',
      status: 500,
    });

    const requests = mockServer.getRequests();
    expect(requests.map((request) => new URL(request.url).pathname)).toEqual([
      '/api/v1/storages',
      '/api/v1/storages',
    ]);

    const queries = requests.map((request) => new URL(request.url).searchParams);
    expect(queries.map((query) => query.get('after'))).toEqual([null, 'second-page']);
    expect(queries.map((query) => query.get('limit'))).toEqual(['100', '100']);
    expect(queries.map((query) => query.get('workspaceId'))).toEqual([
      'workspace-id',
      'workspace-id',
    ]);
  });
});
