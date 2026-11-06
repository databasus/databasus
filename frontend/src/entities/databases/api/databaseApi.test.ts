import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { databaseApi } from '@/entities/databases/api/databaseApi';
import { createDefaultDatabaseSettings } from '@/entities/databases/model/createDefaultDatabaseSettings';
import type {
  DatabaseCreateDraft,
  DatabaseEditDraft,
} from '@/entities/databases/model/DatabaseConnectionDraft';
import { DatabaseType } from '@/entities/databases/model/DatabaseType';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { buildProblem, setUpMockApi } from '@/shared/lib/test/mockApi';

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

describe('databaseApi with connection drafts', () => {
  const savedSettings = {
    ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
    host: 'db.example.com',
    username: 'backup',
    database: 'orders',
    password: undefined,
    sslClientCert: 'saved-client-cert',
  };

  const recordBody = (method: 'post' | 'put', path: string) => {
    const sentBodies: unknown[] = [];
    mockServer.use(
      http[method](`*${API_BASE_PATH}${path}`, async ({ request }) => {
        sentBodies.push(await request.json());
        return HttpResponse.json(buildDatabase('Orders'));
      }),
    );

    return sentBodies;
  };

  it('creates a database with the editable fields of a create draft', async () => {
    const createDraft: DatabaseCreateDraft = {
      scenario: 'create',
      workspaceId: 'workspace-id',
      name: 'Orders',
      notifierIds: ['notifier-id'],
      settings: { ...savedSettings, password: 'typed-secret', sslClientCert: '' },
      hasUserChosenSslMode: true,
    };
    const sentBodies = recordBody('post', '/databases');

    await databaseApi.createDatabase(createDraft);

    expect(sentBodies).toHaveLength(1);
    expect(sentBodies[0]).toMatchObject({
      workspaceId: 'workspace-id',
      name: 'Orders',
      notifierIds: ['notifier-id'],
      settings: { host: 'db.example.com', password: 'typed-secret' },
    });
    expect(sentBodies[0]).not.toHaveProperty('scenario');
    expect(sentBodies[0]).not.toHaveProperty('hasUserChosenSslMode');
    expect((sentBodies[0] as { settings: object }).settings).not.toHaveProperty('sslClientCert');
  });

  it('leaves empty secrets out when updating from an edit draft', async () => {
    const editDraft: DatabaseEditDraft = {
      scenario: 'edit',
      workspaceId: 'workspace-id',
      databaseId: 'database-id',
      name: 'Orders',
      notifierIds: [],
      savedSettings,
      settings: { ...savedSettings, password: '', sslClientCert: '', sslClientKey: '' },
      hasUserChosenSslMode: true,
    };
    const sentBodies = recordBody('put', '/databases/database-id');

    await databaseApi.updateDatabase(editDraft.databaseId, editDraft);

    const sentSettings = (sentBodies[0] as { settings: object }).settings;
    expect(sentBodies[0]).not.toHaveProperty('savedSettings');
    expect(sentBodies[0]).not.toHaveProperty('databaseId');
    expect(sentSettings).not.toHaveProperty('password');
    expect(sentSettings).not.toHaveProperty('sslClientCert');
    expect(sentSettings).not.toHaveProperty('sslClientKey');
  });
});
