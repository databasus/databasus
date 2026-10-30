import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { buildProblem, setUpMockApi } from '../../../shared/lib/test/mockApi';
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
  it('returns the databases of every page when the final page is empty', async () => {
    const requestedQueries: URLSearchParams[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        const databasePage: ApiComponents['schemas']['DatabasePage'] =
          query.get('after') === null
            ? { items: [buildDatabase('Alpha')], nextCursor: 'first-page-cursor' }
            : query.get('after') === 'first-page-cursor'
              ? { items: [buildDatabase('Bravo')], nextCursor: 'last-page-cursor' }
              : { items: [] };

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
      'last-page-cursor',
    ]);
    expect(requestedQueries.map((query) => query.get('notifierId'))).toEqual([
      'notifier-id',
      'notifier-id',
      'notifier-id',
    ]);
    expect(requestedQueries.map((query) => query.get('limit'))).toEqual(['100', '100', '100']);
  });
  it('rejects the complete list when a later page fails without retrying or changing filters', async () => {
    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, ({ request }) => {
        const after = new URL(request.url).searchParams.get('after');
        return after === null
          ? HttpResponse.json({ items: [buildDatabase('Alpha')], nextCursor: 'second-page' })
          : HttpResponse.json(buildProblem('internal', 500), { status: 500 });
      }),
    );
    await expect(
      databaseApi.listAllDatabases('workspace-id', { notifierId: 'notifier-id' }),
    ).rejects.toMatchObject({ code: 'internal', status: 500 });
    const requests = mockServer.getRequests();
    expect(requests.map((request) => new URL(request.url).pathname)).toEqual([
      '/api/v1/databases',
      '/api/v1/databases',
    ]);
    const queries = requests.map((request) => new URL(request.url).searchParams);
    expect(queries.map((query) => query.get('after'))).toEqual([null, 'second-page']);
    expect(queries.map((query) => query.get('limit'))).toEqual(['100', '100']);
    expect(queries.map((query) => query.get('workspaceId'))).toEqual([
      'workspace-id',
      'workspace-id',
    ]);
    expect(queries.map((query) => query.get('notifierId'))).toEqual(['notifier-id', 'notifier-id']);
  });
});
