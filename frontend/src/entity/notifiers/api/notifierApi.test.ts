import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { notifierApi } from './notifierApi';

const mockServer = setUpMockApi();

const buildNotifier = (name: string): ApiComponents['schemas']['Notifier'] => ({
  id: `${name}-id`,
  workspaceId: 'workspace-id',
  name,
  lastSendError: null,
  settings: { type: 'DISCORD' },
});

describe('notifierApi.listAllNotifiers', () => {
  it('returns the notifiers of every page when the list has two pages', async () => {
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
            : { items: [buildNotifier('Bravo')] };

        return HttpResponse.json(notifierPage);
      }),
    );

    const listedNotifiers = await notifierApi.listAllNotifiers('workspace-id');

    expect(listedNotifiers.map((notifier) => notifier.name)).toEqual(['Alpha', 'Bravo']);
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
