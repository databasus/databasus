import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { workspaceApi } from '@/entities/workspaces/api/workspaceApi';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { buildProblem, setUpMockApi } from '@/shared/lib/test/mockApi';

const mockServer = setUpMockApi();

const buildWorkspace = (name: string): ApiComponents['schemas']['Workspace'] => ({
  id: `${name}-id`,
  name,
  createdAt: '2026-10-01T10:00:00Z',
  currentUserRole: 'WORKSPACE_OWNER',
});

describe('workspaceApi.listAllWorkspaces', () => {
  it('returns the workspaces of every page when the final page is empty', async () => {
    const requestedCursors: (string | null)[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/workspaces`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        const workspacePage: ApiComponents['schemas']['WorkspacePage'] =
          cursor === null
            ? { items: [buildWorkspace('Alpha')], nextCursor: 'first-page-cursor' }
            : cursor === 'first-page-cursor'
              ? { items: [buildWorkspace('Bravo')], nextCursor: 'last-page-cursor' }
              : { items: [] };

        return HttpResponse.json(workspacePage);
      }),
    );

    const listedWorkspaces = await workspaceApi.listAllWorkspaces();

    expect(listedWorkspaces.map((workspace) => workspace.name)).toEqual(['Alpha', 'Bravo']);
    expect(requestedCursors).toEqual([null, 'first-page-cursor', 'last-page-cursor']);
  });
  it('rejects the complete list when a later page fails without retrying or changing filters', async () => {
    mockServer.use(
      http.get(`*${API_BASE_PATH}/workspaces`, ({ request }) => {
        const after = new URL(request.url).searchParams.get('after');
        return after === null
          ? HttpResponse.json({ items: [buildWorkspace('Alpha')], nextCursor: 'second-page' })
          : HttpResponse.json(buildProblem('internal', 500), { status: 500 });
      }),
    );

    await expect(workspaceApi.listAllWorkspaces()).rejects.toMatchObject({
      code: 'internal',
      status: 500,
    });

    const requests = mockServer.getRequests();
    expect(requests.map((request) => new URL(request.url).pathname)).toEqual([
      '/api/v1/workspaces',
      '/api/v1/workspaces',
    ]);

    const queries = requests.map((request) => new URL(request.url).searchParams);
    expect(queries.map((query) => query.get('after'))).toEqual([null, 'second-page']);
    expect(queries.map((query) => query.get('limit'))).toEqual(['100', '100']);
  });
});
