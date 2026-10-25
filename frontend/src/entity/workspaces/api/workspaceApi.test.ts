import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { workspaceApi } from './workspaceApi';

const mockServer = setUpMockApi();

const buildWorkspace = (name: string): ApiComponents['schemas']['Workspace'] => ({
  id: `${name}-id`,
  name,
  createdAt: '2026-10-01T10:00:00Z',
  currentUserRole: 'WORKSPACE_OWNER',
});

describe('workspaceApi.listAllWorkspaces', () => {
  it('returns the workspaces of every page when the list has two pages', async () => {
    const requestedCursors: (string | null)[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/workspaces`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        const workspacePage: ApiComponents['schemas']['WorkspacePage'] =
          cursor === null
            ? { items: [buildWorkspace('Alpha')], nextCursor: 'first-page-cursor' }
            : { items: [buildWorkspace('Bravo')] };

        return HttpResponse.json(workspacePage);
      }),
    );

    const listedWorkspaces = await workspaceApi.listAllWorkspaces();

    expect(listedWorkspaces.map((workspace) => workspace.name)).toEqual(['Alpha', 'Bravo']);
    expect(requestedCursors).toEqual([null, 'first-page-cursor']);
  });
});
