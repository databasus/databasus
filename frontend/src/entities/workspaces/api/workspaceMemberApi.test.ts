import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { workspaceMemberApi } from './workspaceMemberApi';

const mockServer = setUpMockApi();

describe('workspaceMemberApi.listAllMembers', () => {
  it('returns the members of every page and sends the workspace, limit and cursor', async () => {
    const requestedQueries: URLSearchParams[] = [];
    mockServer.use(
      http.get('*/api/v1/workspace-members', ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        return HttpResponse.json(
          query.get('after') === null
            ? { items: [{ id: 'first-member' }], nextCursor: 'second-page' }
            : { items: [{ id: 'second-member' }] },
        );
      }),
    );

    const listedMembers = await workspaceMemberApi.listAllMembers('workspace-id');

    expect(listedMembers.map((member) => member.id)).toEqual(['first-member', 'second-member']);
    expect(requestedQueries.map((query) => query.get('after'))).toEqual([null, 'second-page']);
    expect(requestedQueries.map((query) => query.get('workspaceId'))).toEqual([
      'workspace-id',
      'workspace-id',
    ]);
    expect(requestedQueries.map((query) => query.get('limit'))).toEqual(['100', '100']);
  });
});
