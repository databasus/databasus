import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { notifierApi } from '@/entities/notifiers/api/notifierApi';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { buildProblem, setUpMockApi } from '@/shared/lib/test/mockApi';

const mockServer = setUpMockApi();

const buildNotifier = (name: string): ApiComponents['schemas']['Notifier'] => ({
  id: `${name}-id`,
  workspaceId: 'workspace-id',
  name,
  lastSendError: null,
  settings: { type: 'DISCORD' },
});

describe('notifierApi.listAllNotifiers', () => {
  it('returns the notifiers of every page when the final page is empty', async () => {
    const requestedQueries: URLSearchParams[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/notifiers`, ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        const notifierPage: ApiComponents['schemas']['NotifierPage'] =
          query.get('after') === null
            ? {
                items: [buildNotifier('Alpha')],
                nextCursor: 'first-page-cursor',
              }
            : query.get('after') === 'first-page-cursor'
              ? { items: [buildNotifier('Bravo')], nextCursor: 'last-page-cursor' }
              : { items: [] };

        return HttpResponse.json(notifierPage);
      }),
    );

    const listedNotifiers = await notifierApi.listAllNotifiers('workspace-id');

    expect(listedNotifiers.map((notifier) => notifier.name)).toEqual(['Alpha', 'Bravo']);
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
      http.get(`*${API_BASE_PATH}/notifiers`, ({ request }) => {
        const after = new URL(request.url).searchParams.get('after');
        return after === null
          ? HttpResponse.json({ items: [buildNotifier('Alpha')], nextCursor: 'second-page' })
          : HttpResponse.json(buildProblem('internal', 500), { status: 500 });
      }),
    );

    await expect(notifierApi.listAllNotifiers('workspace-id')).rejects.toMatchObject({
      code: 'internal',
      status: 500,
    });

    const requests = mockServer.getRequests();
    expect(requests.map((request) => new URL(request.url).pathname)).toEqual([
      '/api/v1/notifiers',
      '/api/v1/notifiers',
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
