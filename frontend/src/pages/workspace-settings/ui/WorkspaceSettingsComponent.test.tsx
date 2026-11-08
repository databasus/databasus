// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { User } from '@/entities/users';
import type { Workspace } from '@/entities/workspaces';
import { WorkspaceSettingsComponent } from '@/pages/workspace-settings/ui/WorkspaceSettingsComponent';
import { activateSession } from '@/shared/auth';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const workspace: Workspace = {
  id: 'workspace-id',
  name: 'Team',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-10-01T10:00:00Z',
};
const user: User = {
  id: 'owner-id',
  name: 'Owner',
  email: 'owner@example.com',
  role: 'MEMBER',
  status: 'ACTIVE',
  createdAt: '2026-10-01T10:00:00Z',
};
const mockChildren = () =>
  server.use(
    mockApi.get('/workspace-members', 200, { items: [] }),
    mockApi.get('/audit-events', 200, { items: [] }),
  );

describe('WorkspaceSettingsComponent', () => {
  it.each(['Cancel', 'Escape', 'Close'])(
    'sends no deletion when %s dismisses confirmation during the prerequisite read',
    async (dismissal) => {
      const databaseRead = mockApi.pending('get', '/databases');
      server.use(
        mockApi.get('/workspaces/{workspaceId}', 200, workspace),
        databaseRead.handler,
        mockApi.delete('/workspaces/{workspaceId}', 204, undefined),
      );
      mockChildren();
      const onDeleted = vi.fn();
      const view = renderScreen(
        <WorkspaceSettingsComponent
          selectedWorkspace={workspace}
          user={user}
          onDeleted={onDeleted}
        />,
      );
      await screen.findByDisplayValue(workspace.name);

      await view.user.click(screen.getByRole('button', { name: 'Delete workspace' }));
      const dialog = within(await screen.findByRole('dialog'));

      await view.user.click(dialog.getByRole('button', { name: 'Delete Workspace' }));
      await waitFor(() =>
        expect(
          server
            .getRequests()
            .some((request) => new URL(request.url).pathname === '/api/v1/databases'),
        ).toBe(true),
      );

      if (dismissal === 'Escape') fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
      else await view.user.click(dialog.getByRole('button', { name: dismissal }));

      await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

      const prerequisite = server
        .getRequests()
        .find((request) => new URL(request.url).pathname === '/api/v1/databases')!;
      expect(prerequisite.signal.aborted).toBe(true);

      await view.user.click(screen.getByRole('button', { name: 'Delete workspace' }));
      await finishCancelledApiResponse(server, '/api/v1/databases', () =>
        databaseRead.respond(HttpResponse.json({ items: [] })),
      );

      expect(screen.getByRole('dialog')).toBeTruthy();
      expect(server.getRequests().filter((request) => request.method === 'DELETE')).toHaveLength(0);
      expect(onDeleted).not.toHaveBeenCalled();
    },
  );

  it('describes the workspace name error until the draft is corrected', async () => {
    server.use(mockApi.get('/workspaces/{workspaceId}', 200, workspace));
    mockChildren();
    const { user: interaction } = renderScreen(
      <WorkspaceSettingsComponent selectedWorkspace={workspace} user={user} />,
    );
    const name = await screen.findByRole('textbox', { name: 'Workspace name' });

    await interaction.clear(name);
    await interaction.click(screen.getByRole('button', { name: /Save changes/i }));

    const error = screen.getByRole('alert');
    expect(
      screen.getByRole('textbox', { name: 'Workspace name', description: error.textContent! }),
    ).toBe(name);
    expect(name.getAttribute('aria-invalid')).toBe('true');
    expect(server.getRequests().some((request) => request.method === 'PUT')).toBe(false);

    await interaction.type(name, 'Corrected workspace');

    expect(screen.queryByRole('alert')).toBeNull();
    expect(name.getAttribute('aria-invalid')).not.toBe('true');
    expect(name.getAttribute('aria-describedby')).toBeNull();
  });

  it('offers retry after the initial workspace request fails', async () => {
    server.use(mockApi.get('/workspaces/{workspaceId}', 500, buildProblem('internal', 500)));
    mockChildren();
    const { user: interaction } = renderScreen(
      <WorkspaceSettingsComponent selectedWorkspace={workspace} user={user} />,
    );
    const retry = await screen.findByRole('button', { name: 'Retry' });
    server.use(mockApi.get('/workspaces/{workspaceId}', 200, workspace));

    await interaction.click(retry);

    expect(await screen.findByDisplayValue('Team')).toBeTruthy();
  });

  it.each([
    { source: 'member read', status: 403 },
    { source: 'member read', status: 404 },
    { source: 'rename', status: 404 },
  ])(
    'restores settings when retry returns unchanged data after the $source returns HTTP $status',
    async ({ source, status }) => {
      server.use(
        mockApi.get('/workspaces/{workspaceId}', 200, workspace),
        mockApi.put('/workspaces/{workspaceId}', 404, buildProblem('not_found', 404)),
      );
      mockChildren();
      const memberRead =
        source === 'member read' ? mockApi.pending('get', '/workspace-members') : undefined;
      if (memberRead) server.use(memberRead.handler);

      const onUnavailable = vi.fn();
      const onRenamed = vi.fn();
      const getWorkspaceReads = () =>
        server
          .getRequests()
          .filter(
            (request) =>
              request.method === 'GET' &&
              new URL(request.url).pathname === `/api/v1/workspaces/${workspace.id}`,
          );

      const { user: interaction } = renderScreen(
        <WorkspaceSettingsComponent
          selectedWorkspace={workspace}
          user={user}
          onUnavailable={onUnavailable}
          onRenamed={onRenamed}
        />,
      );
      const name = await screen.findByDisplayValue(workspace.name);

      if (memberRead) {
        await waitFor(() =>
          expect(
            server
              .getRequests()
              .some((request) => new URL(request.url).pathname === '/api/v1/workspace-members'),
          ).toBe(true),
        );

        await finishApiResponse(server, '/api/v1/workspace-members', () =>
          memberRead.respond(
            HttpResponse.json(
              buildProblem(status === 403 ? 'permission_denied' : 'not_found', status),
              { status },
            ),
          ),
        );
      } else {
        fireEvent.change(name, { target: { value: 'Submitted name' } });
        await interaction.click(screen.getByRole('button', { name: /Save changes/i }));
      }

      const retry = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.queryByRole('textbox', { name: 'Workspace name' })).toBeNull();
      expect(onUnavailable).toHaveBeenCalledExactlyOnceWith(workspace.id);
      expect(onRenamed).not.toHaveBeenCalled();

      const restoredWorkspace = mockApi.pending('get', '/workspaces/{workspaceId}');
      server.use(restoredWorkspace.handler);
      mockChildren();

      await interaction.click(retry);
      await waitFor(() => expect(getWorkspaceReads()).toHaveLength(2));
      await finishApiResponse(server, `/api/v1/workspaces/${workspace.id}`, () =>
        restoredWorkspace.respond(HttpResponse.json(workspace)),
      );

      expect(screen.getByRole('textbox', { name: 'Workspace name' })).toHaveProperty(
        'value',
        workspace.name,
      );
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(onUnavailable).toHaveBeenCalledExactlyOnceWith(workspace.id);
      expect(onRenamed).not.toHaveBeenCalled();
    },
  );

  it('retains newer name input when the submitted rename finishes', async () => {
    server.use(mockApi.get('/workspaces/{workspaceId}', 200, workspace));
    mockChildren();

    const pendingRename = mockApi.delayed({
      method: 'put',
      path: '/workspaces/{workspaceId}',
      status: 200,
      body: { ...workspace, name: 'Submitted' },
    });
    server.use(pendingRename.handler);

    const onRenamed = vi.fn();
    renderScreen(
      <WorkspaceSettingsComponent
        selectedWorkspace={workspace}
        user={user}
        onRenamed={onRenamed}
      />,
    );
    const name = await screen.findByDisplayValue('Team');

    fireEvent.change(name, { target: { value: 'Submitted' } });
    fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));

    await waitFor(() =>
      expect(server.getRequests().some((request) => request.method === 'PUT')).toBe(true),
    );

    fireEvent.change(name, { target: { value: 'Newer' } });
    pendingRename.release();
    await screen.findByText('Basic information updated successfully');

    expect(screen.getByDisplayValue('Newer')).toBeTruthy();
    expect(onRenamed).toHaveBeenCalledWith({ ...workspace, name: 'Submitted' });
    expect(screen.getByRole('button', { name: /Save changes/i })).toBeTruthy();
  });
  it.each([403, 404, 500])(
    'reconciles access loss and retains recovery when workspace reads return HTTP %s',
    async (status) => {
      server.use(
        mockApi.get(
          '/workspaces/{workspaceId}',
          status,
          buildProblem(
            status === 404 ? 'not_found' : status === 403 ? 'permission_denied' : 'internal',
            status,
          ),
        ),
      );
      const onUnavailable = vi.fn();
      renderScreen(
        <WorkspaceSettingsComponent
          selectedWorkspace={workspace}
          user={user}
          onUnavailable={onUnavailable}
        />,
        { isStrictMode: true },
      );
      await screen.findByRole('button', { name: 'Retry' });
      if (status === 500) expect(onUnavailable).not.toHaveBeenCalled();
      else expect(onUnavailable).toHaveBeenCalledWith(workspace.id);

      expect(screen.queryByDisplayValue('Team')).toBeNull();
    },
  );

  it('offers read-only membership and no workspace deletion when the member is a viewer', async () => {
    const viewerWorkspace: Workspace = { ...workspace, currentUserRole: 'WORKSPACE_VIEWER' };
    server.use(mockApi.get('/workspaces/{workspaceId}', 200, viewerWorkspace));
    mockChildren();
    renderScreen(<WorkspaceSettingsComponent selectedWorkspace={viewerWorkspace} user={user} />);

    expect(
      ((await screen.findByRole('textbox', { name: 'Workspace name' })) as HTMLInputElement)
        .disabled,
    ).toBe(true);
    expect(screen.queryByRole('button', { name: /Delete workspace/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /Add member/i })).toBeNull();
  });

  it('deletes only once when confirmation repeats while the prerequisite database read is pending', async () => {
    const databaseRead = mockApi.pending('get', '/databases');
    const deletion = mockApi.pending('delete', '/workspaces/{workspaceId}');
    server.use(
      mockApi.get('/workspaces/{workspaceId}', 200, workspace),
      databaseRead.handler,
      deletion.handler,
    );
    mockChildren();
    const onDeleted = vi.fn();
    const { user: interaction } = renderScreen(
      <WorkspaceSettingsComponent
        selectedWorkspace={workspace}
        user={user}
        onDeleted={onDeleted}
      />,
    );
    await screen.findByDisplayValue(workspace.name);

    await interaction.click(screen.getByRole('button', { name: 'Delete workspace' }));
    const dialog = within(await screen.findByRole('dialog'));
    const confirm = dialog.getByRole('button', { name: 'Delete Workspace' });

    act(() => {
      fireEvent.click(confirm);
      fireEvent.click(confirm);
    });
    await waitFor(() =>
      expect(
        server
          .getRequests()
          .filter((request) => new URL(request.url).pathname === '/api/v1/databases'),
      ).toHaveLength(1),
    );

    expect(confirm.getAttribute('aria-busy')).toBe('true');
    expect((confirm as HTMLButtonElement).disabled).toBe(true);
    expect(server.getRequests().filter((request) => request.method === 'DELETE')).toHaveLength(0);

    await interaction.click(confirm);
    await interaction.click(confirm);
    await finishApiResponse(server, '/api/v1/databases', () =>
      databaseRead.respond(HttpResponse.json({ items: [] })),
    );
    await waitFor(() =>
      expect(server.getRequests().filter((request) => request.method === 'DELETE')).toHaveLength(1),
    );

    expect(confirm.getAttribute('aria-busy')).toBe('true');
    expect((confirm as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByRole('dialog')).toBeTruthy();
    expect(onDeleted).not.toHaveBeenCalled();

    await interaction.click(confirm);
    await finishApiResponse(server, `/api/v1/workspaces/${workspace.id}`, () =>
      deletion.respond(new HttpResponse(null, { status: 204 })),
    );

    expect(onDeleted).toHaveBeenCalledExactlyOnceWith(workspace.id);
    expect(server.getRequests().filter((request) => request.method === 'DELETE')).toHaveLength(1);
    expect(
      server
        .getRequests()
        .filter((request) => new URL(request.url).pathname === '/api/v1/databases'),
    ).toHaveLength(1);
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(screen.getAllByText('Workspace deleted successfully')).toHaveLength(1);
  });

  it('keeps the workspace and explains individual storage deletion when cleanup is pending', async () => {
    server.use(
      mockApi.get('/workspaces/{workspaceId}', 200, workspace),
      mockApi.get('/databases', 200, { items: [] }),
      mockApi.delete('/workspaces/{workspaceId}', 409, buildProblem('cleanup_pending', 409)),
    );
    mockChildren();
    const onDeleted = vi.fn();
    renderScreen(
      <WorkspaceSettingsComponent
        selectedWorkspace={workspace}
        user={user}
        onDeleted={onDeleted}
      />,
    );
    await screen.findByDisplayValue(workspace.name);
    fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Delete Workspace' }),
    );

    await screen.findByText(
      'File cleanup is still running. Wait for it to finish, or delete the affected storages individually and acknowledge that any remaining files must be removed manually.',
    );

    expect(screen.getByDisplayValue(workspace.name)).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Delete anyway/i })).toBeNull();
    expect(onDeleted).not.toHaveBeenCalled();
    const deletions = server.getRequests().filter((request) => request.method === 'DELETE');
    expect(deletions).toHaveLength(1);
    expect(new URL(deletions[0].url).searchParams.has('shouldAbandonCleanup')).toBe(false);
  });

  it.each(['rename', 'delete'] as const)(
    'reconciles workspace unavailability when %s returns 404',
    async (operation) => {
      server.use(
        mockApi.get('/workspaces/{workspaceId}', 200, workspace),
        mockApi.get('/databases', 200, { items: [] }),
        operation === 'rename'
          ? mockApi.put('/workspaces/{workspaceId}', 404, buildProblem('not_found', 404))
          : mockApi.delete('/workspaces/{workspaceId}', 404, buildProblem('not_found', 404)),
      );
      mockChildren();

      const onUnavailable = vi.fn();
      const onRenamed = vi.fn();
      const onDeleted = vi.fn();
      renderScreen(
        <WorkspaceSettingsComponent
          selectedWorkspace={workspace}
          user={user}
          onUnavailable={onUnavailable}
          onRenamed={onRenamed}
          onDeleted={onDeleted}
        />,
      );
      const name = await screen.findByDisplayValue(workspace.name);

      if (operation === 'rename') {
        fireEvent.change(name, { target: { value: 'Renamed' } });
        fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));
      } else {
        fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
        fireEvent.click(
          within(await screen.findByRole('dialog')).getByRole('button', {
            name: 'Delete Workspace',
          }),
        );
      }

      await waitFor(() => expect(onUnavailable).toHaveBeenCalledWith(workspace.id));
      expect(screen.queryByRole('textbox', { name: 'Workspace name' })).toBeNull();
      expect(onRenamed).not.toHaveBeenCalled();
      expect(onDeleted).not.toHaveBeenCalled();
    },
  );

  it.each(
    (['read', 'rename', 'delete'] as const).flatMap((operation) =>
      (['session', 'unmount'] as const).flatMap((lifetime) =>
        [false, true].map((hasFailed) => ({ operation, lifetime, hasFailed })),
      ),
    ),
  )(
    'suppresses obsolete $operation responses after $lifetime with failure=$hasFailed',
    async ({ operation, lifetime, hasFailed }) => {
      activateSession({ token: `initial-${lifetime}-${hasFailed}`, userId: user.id });
      const pending =
        operation === 'read'
          ? mockApi.pending('get', '/workspaces/{workspaceId}')
          : operation === 'rename'
            ? mockApi.pending('put', '/workspaces/{workspaceId}')
            : mockApi.pending('delete', '/workspaces/{workspaceId}');

      server.use(
        mockApi.get('/workspaces/{workspaceId}', 200, workspace),
        mockApi.get('/databases', 200, { items: [] }),
      );
      server.use(pending.handler);
      mockChildren();

      const onRenamed = vi.fn();
      const onDeleted = vi.fn();
      const onUnavailable = vi.fn();
      const screenProps = { user, onRenamed, onDeleted, onUnavailable };
      const view = renderScreen(
        <WorkspaceSettingsComponent {...screenProps} selectedWorkspace={workspace} />,
        { isStrictMode: true },
      );

      if (operation !== 'read') {
        const name = await screen.findByDisplayValue(workspace.name);
        if (operation === 'rename') {
          fireEvent.change(name, { target: { value: 'Obsolete rename' } });
          fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));
        } else {
          fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
          fireEvent.click(
            within(await screen.findByRole('dialog')).getByRole('button', {
              name: 'Delete Workspace',
            }),
          );
        }
      }

      const method = operation === 'read' ? 'GET' : operation === 'rename' ? 'PUT' : 'DELETE';

      await waitFor(() =>
        expect(
          server
            .getRequests()
            .some(
              (request) =>
                request.method === method &&
                new URL(request.url).pathname.endsWith(`/workspaces/${workspace.id}`),
            ),
        ).toBe(true),
      );

      if (lifetime === 'session') {
        act(() => activateSession({ token: 'replacement-token', userId: 'replacement-user' }));
      } else view.unmount();

      const finishResponse = operation === 'read' ? finishCancelledApiResponse : finishApiResponse;
      await finishResponse(server, `/api/v1/workspaces/${workspace.id}`, () => {
        pending.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('not_found', 404), { status: 404 })
            : operation === 'delete'
              ? new HttpResponse(null, { status: 204 })
              : HttpResponse.json({ ...workspace, name: 'Obsolete response' }),
        );
      });

      expect(onRenamed).not.toHaveBeenCalled();
      expect(onDeleted).not.toHaveBeenCalled();
      expect(onUnavailable).not.toHaveBeenCalled();
      expect(screen.queryByDisplayValue('Obsolete response')).toBeNull();
      expect(screen.queryByText('Basic information updated successfully')).toBeNull();
      expect(screen.queryByText('Workspace deleted successfully')).toBeNull();
      expect(
        screen.queryByText('The requested item was not found. It may have been deleted.'),
      ).toBeNull();
      expect(view.container.querySelector('[role="alert"]')).toBeNull();

      view.unmount();
    },
  );

  it.each([403, 500])(
    'retains the workspace name draft when a rename returns HTTP %s',
    async (status) => {
      server.use(
        mockApi.get('/workspaces/{workspaceId}', 200, workspace),
        mockApi.put(
          '/workspaces/{workspaceId}',
          status,
          buildProblem(status === 403 ? 'permission_denied' : 'internal', status),
        ),
      );
      mockChildren();
      const onUnavailable = vi.fn();
      const onRenamed = vi.fn();
      renderScreen(
        <WorkspaceSettingsComponent
          selectedWorkspace={workspace}
          user={user}
          onUnavailable={onUnavailable}
          onRenamed={onRenamed}
        />,
      );

      fireEvent.change(await screen.findByDisplayValue(workspace.name), {
        target: { value: 'Retained draft' },
      });

      fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));

      await screen.findByText(
        status === 403
          ? 'You do not have permission to do this.'
          : 'An internal server error occurred. Please try again later.',
      );

      expect(screen.getByDisplayValue('Retained draft')).toBeTruthy();
      expect(screen.getByRole('button', { name: /Save changes/i })).toHaveProperty(
        'disabled',
        false,
      );
      expect(onUnavailable).not.toHaveBeenCalled();
      expect(onRenamed).not.toHaveBeenCalled();
    },
  );
});
