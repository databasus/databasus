import { describe, expect, it } from 'vitest';

import { inspectApiRequest } from './inspectApiRequest';

describe('inspectApiRequest', () => {
  it('inspects the declared path, query and body of outgoing requests', async () => {
    const request = new Request('http://localhost/api/v1/workspaces/workspace-id?unused=1');
    const workspace = await inspectApiRequest(request, {
      method: 'get',
      path: '/workspaces/{workspaceId}',
    });
    expect(workspace.params.workspaceId).toBe('workspace-id');

    const createWorkspace = await inspectApiRequest(
      new Request('http://localhost/api/v1/workspaces', {
        method: 'POST',
        body: JSON.stringify({ name: 'Research' }),
      }),
      { method: 'post', path: '/workspaces' },
    );
    expect(createWorkspace.body.name).toBe('Research');

    const databases = await inspectApiRequest(
      new Request('http://localhost/api/v1/databases?workspaceId=workspace-id'),
      { method: 'get', path: '/databases' },
    );
    expect(databases.query.get('workspaceId')).toBe('workspace-id');

    await expect(
      inspectApiRequest(request, { method: 'get', path: '/system/version' }),
    ).rejects.toThrow('Request does not match');
  });
});
