// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { Workspace } from '@/entities/workspaces';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';

import { renderApp } from './renderApp';

const server = setUpMockApi();
const firstWorkspace: Workspace = {
  id: 'first-workspace',
  name: 'First team',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-10-01T00:00:00Z',
};
const secondWorkspace: Workspace = {
  ...firstWorkspace,
  id: 'second-workspace',
  name: 'Second team',
};
const thirdWorkspace: Workspace = { ...firstWorkspace, id: 'third-workspace', name: 'Third team' };
const openApplication = (workspaces: Workspace[], rememberedWorkspaceId?: string) => {
  activateSession({ token: 'workspace-session', userId: 'owner-id' });

  if (rememberedWorkspaceId) localStorage.setItem('selected_workspace_id', rememberedWorkspaceId);

  server.use(
    mockApi.get('/system/disk-usages', 200, {
      usedSpaceBytes: 1,
      totalSpaceBytes: 10,
      freeSpaceBytes: 9,
    }),
    mockApi.get('/users/{userId}', 200, {
      id: 'owner-id',
      name: 'Owner',
      email: 'owner@example.com',
      role: 'ADMIN',
      status: 'ACTIVE',
      createdAt: firstWorkspace.createdAt,
    }),
    mockApi.get('/instance-settings', 200, {
      isAllowExternalRegistrations: false,
      isAllowMemberInvitations: false,
      isMemberAllowedToCreateWorkspaces: false,
      isEmailConfigured: true,
      isTwoFactorAuthRequired: false,
    }),
    mockApi.get('/workspaces', 200, { items: workspaces }),
    http.get(`*${API_BASE_PATH}/workspaces/:workspaceId`, ({ params }) =>
      HttpResponse.json(workspaces.find((workspace) => workspace.id === params.workspaceId)),
    ),
    mockApi.get('/databases', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/storage-switches', 200, { items: [] }),
    mockApi.get('/workspace-members', 200, { items: [] }),
    mockApi.get('/audit-events', 200, { items: [] }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'v0.0.0' }),
    ),
  );

  return renderApp();
};
const getWorkspaceSelection = () =>
  within(
    screen
      .getByRole('combobox', { name: 'Selected workspace' })
      .closest('.ant-select') as HTMLElement,
  );
const selectWorkspace = async (workspace: Workspace) => {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Selected workspace' }));
  const option = await waitFor(() => {
    const option = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
      (option) => option.textContent === workspace.name,
    );

    expect(option).toBeTruthy();

    return option!;
  });

  fireEvent.click(option);
};
const openSettings = async () => {
  fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));
  await screen.findByRole('textbox', { name: 'Workspace name' });
};
const denyFirstWorkspace = () =>
  server.use(
    http.get(`*${API_BASE_PATH}/notifiers`, ({ request }) =>
      new URL(request.url).searchParams.get('workspaceId') === firstWorkspace.id
        ? HttpResponse.json(buildProblem('not_found', 404), { status: 404 })
        : HttpResponse.json({ items: [] }),
    ),
  );

describe('Application workspace navigation', () => {
  it.each(
    (['/audit-events', '/workspace-members'] as const).flatMap((path) =>
      [403, 404, 500].map((status) => ({
        path,
        status,
        scenario:
          status === 500
            ? 'retains workspace input when a child read fails'
            : 'reconciles workspace access when a child read refuses access',
      })),
    ),
  )('$scenario for $path with HTTP $status', async ({ path, status }) => {
    openApplication([firstWorkspace, secondWorkspace]);
    const childRead = mockApi.pending('get', path);
    server.use(childRead.handler);
    await openSettings();
    const name = await screen.findByDisplayValue(firstWorkspace.name);

    await waitFor(() =>
      expect(
        server.getRequests().some((request) => new URL(request.url).pathname.endsWith(path)),
      ).toBe(true),
    );

    fireEvent.change(name, { target: { value: 'Obsolete workspace draft' } });
    server.use(
      mockApi.get('/workspaces', 200, { items: [secondWorkspace] }),
      mockApi.get(path, 200, { items: [] }),
    );

    const previousWorkspaceReads = server
      .getRequests()
      .filter((request) => new URL(request.url).pathname.endsWith('/workspaces')).length;

    childRead.respond(
      HttpResponse.json(
        buildProblem(
          status === 404 ? 'not_found' : status === 403 ? 'permission_denied' : 'internal',
          status,
        ),
        { status },
      ),
    );

    if (status === 500) {
      await screen.findByText('An internal server error occurred. Please try again later.');

      expect(screen.getByDisplayValue('Obsolete workspace draft')).toBeTruthy();
      expect(getWorkspaceSelection().getByText(firstWorkspace.name)).toBeTruthy();
      expect(localStorage.getItem('selected_workspace_id')).toBe(firstWorkspace.id);
      expect(
        server
          .getRequests()
          .filter((request) => new URL(request.url).pathname.endsWith('/workspaces')),
      ).toHaveLength(previousWorkspaceReads);

      return;
    }

    await waitFor(() =>
      expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
    );
    expect(await screen.findByDisplayValue(secondWorkspace.name)).toBeTruthy();
    expect(screen.queryByDisplayValue('Obsolete workspace draft')).toBeNull();
    expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
    expect(localStorage.getItem('selected_workspace_id')).toBe(secondWorkspace.id);
    expect(server.getRequests().some((request) => ['PUT', 'DELETE'].includes(request.method))).toBe(
      false,
    );
  });

  it('restores the remembered workspace after application reload', async () => {
    const application = openApplication([firstWorkspace, secondWorkspace], secondWorkspace.id);
    await screen.findByRole('combobox', { name: 'Selected workspace' });

    expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();

    application.unmount();
    renderApp();
    await screen.findByRole('combobox', { name: 'Selected workspace' });

    expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
    expect(localStorage.getItem('selected_workspace_id')).toBe(secondWorkspace.id);
  });

  it('updates navigation when a workspace rename is acknowledged', async () => {
    openApplication([firstWorkspace, secondWorkspace]);
    server.use(
      mockApi.put('/workspaces/{workspaceId}', 200, { ...firstWorkspace, name: 'Renamed team' }),
    );

    await openSettings();

    fireEvent.change(screen.getByRole('textbox', { name: 'Workspace name' }), {
      target: { value: 'Renamed team' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));

    await waitFor(() => expect(getWorkspaceSelection().getByText('Renamed team')).toBeTruthy());
  });

  it.each([true, false])(
    'deletes the selected workspace without reloading with another workspace=%s',
    async (hasAnotherWorkspace) => {
      const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
      openApplication(hasAnotherWorkspace ? [firstWorkspace, secondWorkspace] : [firstWorkspace]);
      server.use(mockApi.delete('/workspaces/{workspaceId}', 204, undefined));

      await openSettings();

      fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
      fireEvent.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Delete Workspace' }),
      );

      await screen.findByText('Workspace deleted successfully');
      if (hasAnotherWorkspace) {
        await waitFor(() =>
          expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
        );
        expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
      } else {
        await waitFor(() =>
          expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull(),
        );
        expect(screen.getAllByRole('button', { name: 'Create workspace' }).length).toBeGreaterThan(
          0,
        );
        expect(screen.queryByRole('textbox', { name: 'Workspace name' })).toBeNull();
      }

      expect(reload).not.toHaveBeenCalled();
    },
  );

  it.each([false, true])(
    'retains a newer workspace draft after obsolete access reconciliation with failure=%s',
    async (hasFailed) => {
      openApplication([firstWorkspace, secondWorkspace, thirdWorkspace]);
      await screen.findByRole('combobox', { name: 'Selected workspace' });
      const reconciliation = mockApi.pending('get', '/workspaces');
      server.use(reconciliation.handler);
      denyFirstWorkspace();

      fireEvent.click(screen.getByRole('button', { name: 'Notifiers' }));

      await waitFor(() =>
        expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
      );
      await waitFor(() =>
        expect(
          server
            .getRequests()
            .filter((request) => new URL(request.url).pathname.endsWith('/workspaces')),
        ).toHaveLength(2),
      );

      await selectWorkspace(thirdWorkspace);
      await openSettings();
      const name = await screen.findByDisplayValue(thirdWorkspace.name);

      fireEvent.change(name, { target: { value: 'Current workspace draft' } });

      const processed = new Promise<void>((resolve) => {
        server.events.on('response:mocked', function onResponse({ request }) {
          if (!new URL(request.url).pathname.endsWith('/workspaces')) return;

          server.events.removeListener('response:mocked', onResponse);
          resolve();
        });
      });

      await act(async () => {
        reconciliation.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({ items: [secondWorkspace] }),
        );
        await processed;
      });

      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull();
      expect(getWorkspaceSelection().getByText(thirdWorkspace.name)).toBeTruthy();
      expect(screen.getByDisplayValue('Current workspace draft')).toBeTruthy();
      expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
      expect(localStorage.getItem('selected_workspace_id')).toBe(thirdWorkspace.id);
    },
  );

  it('recovers a failed accessible-workspace refresh without rendering unavailable content', async () => {
    openApplication([firstWorkspace, secondWorkspace]);
    await screen.findByRole('combobox', { name: 'Selected workspace' });
    denyFirstWorkspace();
    server.use(mockApi.get('/workspaces', 500, buildProblem('internal', 500)));

    fireEvent.click(screen.getByRole('button', { name: 'Notifiers' }));
    const retry = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull();
    expect(screen.queryByText('No notifiers in this workspace yet')).toBeNull();

    server.use(mockApi.get('/workspaces', 200, { items: [secondWorkspace] }));

    fireEvent.click(retry);
    await screen.findByText('No notifiers in this workspace yet');

    expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it.each(
    (['create', 'rename', 'delete'] as const).flatMap((mutation) =>
      [false, true].map((hasFailed) => ({ mutation, hasFailed })),
    ),
  )(
    'retains an acknowledged workspace $mutation after older reconciliation finishes with failure=$hasFailed',
    async ({ mutation, hasFailed }) => {
      openApplication([firstWorkspace, secondWorkspace]);
      await screen.findByRole('combobox', { name: 'Selected workspace' });
      const reconciliation = mockApi.pending('get', '/workspaces');
      server.use(reconciliation.handler);
      denyFirstWorkspace();

      fireEvent.click(screen.getByRole('button', { name: 'Notifiers' }));

      await waitFor(() =>
        expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
      );

      if (mutation === 'create') {
        server.use(mockApi.post('/workspaces', 201, thirdWorkspace));

        fireEvent.click(
          within(screen.getByRole('banner')).getByRole('button', {
            name: /Create workspace|Create new workspace/,
          }),
        );
        const dialog = within(await screen.findByRole('dialog'));

        fireEvent.change(dialog.getByRole('textbox', { name: 'Workspace name' }), {
          target: { value: thirdWorkspace.name },
        });
        fireEvent.click(dialog.getByRole('button', { name: 'Create workspace' }));

        await waitFor(() =>
          expect(getWorkspaceSelection().getByText(thirdWorkspace.name)).toBeTruthy(),
        );
      } else {
        await openSettings();
        if (mutation === 'rename') {
          server.use(
            mockApi.put('/workspaces/{workspaceId}', 200, {
              ...secondWorkspace,
              name: 'Renamed team',
            }),
          );

          fireEvent.change(screen.getByRole('textbox', { name: 'Workspace name' }), {
            target: { value: 'Renamed team' },
          });
          fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));

          await waitFor(() =>
            expect(getWorkspaceSelection().getByText('Renamed team')).toBeTruthy(),
          );
        } else {
          server.use(mockApi.delete('/workspaces/{workspaceId}', 204, undefined));

          fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
          fireEvent.click(
            within(await screen.findByRole('dialog')).getByRole('button', {
              name: 'Delete Workspace',
            }),
          );

          await waitFor(() =>
            expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull(),
          );
        }
      }

      const processed = new Promise<void>((resolve) => {
        server.events.on('response:mocked', function onResponse() {
          server.events.removeListener('response:mocked', onResponse);
          resolve();
        });
      });

      await act(async () => {
        reconciliation.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({ items: [secondWorkspace] }),
        );
        await processed;
      });

      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull();

      if (mutation === 'delete') {
        expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull();
        expect(localStorage.getItem('selected_workspace_id')).toBeNull();
      } else {
        expect(
          getWorkspaceSelection().getByText(
            mutation === 'create' ? thirdWorkspace.name : 'Renamed team',
          ),
        ).toBeTruthy();
        expect(localStorage.getItem('selected_workspace_id')).toBe(
          mutation === 'create' ? thirdWorkspace.id : secondWorkspace.id,
        );
      }
    },
  );
});
