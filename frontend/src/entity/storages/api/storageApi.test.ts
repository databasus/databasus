import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { storageApi } from './storageApi';

const mockServer = setUpMockApi();

const buildStorage = (name: string): ApiComponents['schemas']['Storage'] => ({
  id: `${name}-id`,
  workspaceId: 'workspace-id',
  name,
  settings: { type: 'LOCAL' },
});

describe('storageApi.listAllStorages', () => {
  it('returns the storages of every page when the list has two pages', async () => {
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
            : { items: [buildStorage('Bravo')] };

        return HttpResponse.json(storagePage);
      }),
    );

    const listedStorages = await storageApi.listAllStorages('workspace-id');

    expect(listedStorages.map((storage) => storage.name)).toEqual(['Alpha', 'Bravo']);
    expect(requestedQueries.map((query) => query.get('after'))).toEqual([
      null,
      'first-page-cursor',
    ]);
    expect(requestedQueries.map((query) => query.get('workspaceId'))).toEqual([
      'workspace-id',
      'workspace-id',
    ]);
  });
});
