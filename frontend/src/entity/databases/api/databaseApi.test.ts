import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { createDefaultDatabaseSettings } from '../model/createDefaultDatabaseSettings';
import { DatabaseType } from '../model/DatabaseType';
import { databaseApi } from './databaseApi';

const mockServer = setUpMockApi();

const buildDatabase = (name: string): ApiComponents['schemas']['Database'] => ({
  id: `${name}-id`,
  workspaceId: 'workspace-id',
  name,
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
});

describe('databaseApi.listAllDatabases', () => {
  it('returns the databases of every page when the list has two pages', async () => {
    const requestedQueries: URLSearchParams[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        const databasePage: ApiComponents['schemas']['DatabasePage'] =
          query.get('after') === null
            ? { items: [buildDatabase('Alpha')], nextCursor: 'first-page-cursor' }
            : { items: [buildDatabase('Bravo')] };

        return HttpResponse.json(databasePage);
      }),
    );

    const listedDatabases = await databaseApi.listAllDatabases('workspace-id', {
      notifierId: 'notifier-id',
    });

    expect(listedDatabases.map((database) => database.name)).toEqual(['Alpha', 'Bravo']);
    expect(requestedQueries.map((query) => query.get('after'))).toEqual([
      null,
      'first-page-cursor',
    ]);
    expect(requestedQueries.map((query) => query.get('notifierId'))).toEqual([
      'notifier-id',
      'notifier-id',
    ]);
    expect(requestedQueries.map((query) => query.get('limit'))).toEqual(['100', '100']);
  });
});
