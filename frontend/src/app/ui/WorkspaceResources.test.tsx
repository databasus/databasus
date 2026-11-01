// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { Workspace } from '@/entities/workspaces';
import { NotifiersComponent } from '@/pages/notifiers';
import { StoragesComponent } from '@/pages/storages';
import { API_BASE_PATH } from '@/shared/api';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const workspace: Workspace = {
  id: 'workspace-a',
  name: 'Team A',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-10-01T00:00:00Z',
};

for (const domain of ['storages', 'notifiers'] as const) {
  const ResourceScreen = ({
    workspace,
    onUnavailable,
  }: {
    workspace: Workspace;
    onUnavailable?: (id: string) => void;
  }) =>
    domain === 'storages' ? (
      <StoragesComponent workspace={workspace} isCanManageStorages onUnavailable={onUnavailable} />
    ) : (
      <NotifiersComponent
        workspace={workspace}
        isCanManageNotifiers
        onUnavailable={onUnavailable}
      />
    );
  const emptyText =
    domain === 'storages'
      ? 'No storages in this workspace yet'
      : 'No notifiers in this workspace yet';

  describe(`${domain} workspace ownership`, () => {
    it.each(
      (['delete', 'transfer'] as const).flatMap((operation) =>
        [false, true].map((isLastResource) => ({ operation, isLastResource })),
      ),
    )(
      'reconciles selection and preference when $operation succeeds with last resource=$isLastResource',
      async ({ operation, isLastResource }) => {
        const resources = ['first', 'second'].map((name) => ({
          id: `${name}-resource`,
          workspaceId: workspace.id,
          name: `Archive ${name}`,
          settings: domain === 'storages' ? { type: 'LOCAL' } : { type: 'DISCORD' },
          ...(domain === 'notifiers' ? { lastSendError: null } : {}),
        }));
        const removedResource = resources[1];
        let listedResources = isLastResource ? [removedResource] : resources;
        const mutation = createPendingApiResponse();
        server.use(
          http.get(`*${API_BASE_PATH}/${domain}`, () =>
            HttpResponse.json({ items: listedResources }),
          ),
          http.get(`*${API_BASE_PATH}/${domain}/:resourceId`, ({ params }) =>
            HttpResponse.json(resources.find((resource) => resource.id === params.resourceId)),
          ),
          http[operation === 'delete' ? 'delete' : 'post'](
            `*${API_BASE_PATH}/${domain}/:resourceId${operation === 'transfer' ? '/transfer' : ''}`,
            () => mutation.response,
          ),
          mockApi.get('/backup-configs', 200, { items: [] }),
          mockApi.get('/databases', 200, { items: [] }),
          mockApi.get('/workspaces', 200, {
            items: [workspace, { ...workspace, id: 'target-id', name: 'Target workspace' }],
          }),
        );

        const selectionKey = `${domain === 'storages' ? 'selectedStorageId' : 'selectedNotifierId'}_${workspace.id}`;
        const view = renderScreen(<ResourceScreen workspace={workspace} />);

        await view.user.click(await screen.findByRole('button', { name: removedResource.name }));

        await waitFor(() => expect(localStorage.getItem(selectionKey)).toBe(removedResource.id));

        if (operation === 'delete') {
          await view.user.click(await screen.findByRole('button', { name: 'Remove' }));
          await view.user.click(
            within(screen.getByRole('dialog')).getByRole('button', { name: 'Remove' }),
          );
        } else {
          await view.user.click(
            await screen.findByRole('button', {
              name: domain === 'storages' ? 'Transfer storage to another workspace' : 'Transfer',
            }),
          );
          const targetSelector = await screen.findByRole('combobox', { name: 'Target workspace' });

          fireEvent.mouseDown(targetSelector);
          const targetOption = await screen.findByText('Target workspace', {
            selector: '.ant-select-item-option-content',
          });

          fireEvent.click(targetOption);
          await view.user.click(
            within(screen.getByRole('dialog')).getByRole('button', { name: 'Transfer' }),
          );
        }

        await waitFor(() =>
          expect(
            server
              .getRequests()
              .some(
                (request) =>
                  request.method === (operation === 'delete' ? 'DELETE' : 'POST') &&
                  new URL(request.url).pathname.endsWith(
                    `/${domain}/${removedResource.id}${operation === 'transfer' ? '/transfer' : ''}`,
                  ),
              ),
          ).toBe(true),
        );
        expect(localStorage.getItem(selectionKey)).toBe(removedResource.id);

        listedResources = isLastResource ? [] : [resources[0]];

        mutation.respond(new HttpResponse(null, { status: 204 }));

        if (isLastResource) {
          await screen.findByText(emptyText);

          await waitFor(() => expect(localStorage.getItem(selectionKey)).toBeNull());
        } else {
          await waitFor(() =>
            expect(
              screen.getByRole('button', { name: resources[0].name }).getAttribute('aria-pressed'),
            ).toBe('true'),
          );
          await waitFor(() => expect(localStorage.getItem(selectionKey)).toBe(resources[0].id));
        }

        expect(screen.queryByRole('button', { name: removedResource.name })).toBeNull();
        expect(view.container.querySelector('.ant-alert-error')).toBeNull();
      },
    );

    it.each([false, true])(
      'retains current resources after an old workspace list finishes with failure=%s',
      async (hasFailed) => {
        const oldList =
          domain === 'storages'
            ? mockApi.pending('get', '/storages')
            : mockApi.pending('get', '/notifiers');
        const currentWorkspace = { ...workspace, id: 'workspace-b', name: 'Team B' };
        const currentStorages = ['first', 'second'].map((name) => ({
          id: `${name}-resource`,
          workspaceId: currentWorkspace.id,
          name: `Current ${name}`,
          settings: { type: 'LOCAL' as const },
        }));
        const currentNotifiers = currentStorages.map((storage) => ({
          ...storage,
          settings: { type: 'DISCORD' as const },
          lastSendError: null,
        }));
        const currentItems = domain === 'storages' ? currentStorages : currentNotifiers;

        server.use(oldList.handler);
        const onUnavailable = vi.fn();
        const view = renderScreen(
          <ResourceScreen workspace={workspace} onUnavailable={onUnavailable} />,
          { isStrictMode: true },
        );

        await waitFor(() =>
          expect(
            server
              .getRequests()
              .some(
                (request) => new URL(request.url).searchParams.get('workspaceId') === workspace.id,
              ),
          ).toBe(true),
        );

        server.use(
          domain === 'storages'
            ? mockApi.get('/storages', 200, { items: currentStorages })
            : mockApi.get('/notifiers', 200, { items: currentNotifiers }),
        );
        server.use(
          http.get(`*${API_BASE_PATH}/${domain}/:resourceId`, ({ params }) =>
            HttpResponse.json(currentItems.find((resource) => resource.id === params.resourceId)),
          ),
        );

        view.rerender(
          <ResourceScreen workspace={currentWorkspace} onUnavailable={onUnavailable} />,
        );
        await view.user.click(await screen.findByRole('button', { name: currentItems[1].name }));

        await waitFor(() =>
          expect(
            screen.getByRole('button', { name: currentItems[1].name }).getAttribute('aria-pressed'),
          ).toBe('true'),
        );

        const selectionKey =
          domain === 'storages'
            ? `selectedStorageId_${currentWorkspace.id}`
            : `selectedNotifierId_${currentWorkspace.id}`;
        const currentPreference = localStorage.getItem(selectionKey);

        expect(currentPreference).toBe(currentItems[1].id);

        const processed = new Promise<void>((resolve) => {
          server.events.on('response:mocked', function onResponse() {
            server.events.removeListener('response:mocked', onResponse);
            resolve();
          });
        });

        await act(async () => {
          oldList.respond(
            hasFailed
              ? HttpResponse.json(buildProblem('not_found', 404), { status: 404 })
              : HttpResponse.json({
                  items: [
                    {
                      ...currentItems[0],
                      id: 'obsolete-resource',
                      workspaceId: workspace.id,
                      name: 'Obsolete resource',
                    },
                  ],
                }),
          );
          await processed;
        });

        for (const resource of currentItems)
          expect(screen.getByRole('button', { name: resource.name })).toBeTruthy();

        expect(
          screen.getByRole('button', { name: currentItems[1].name }).getAttribute('aria-pressed'),
        ).toBe('true');
        expect(localStorage.getItem(selectionKey)).toBe(currentPreference);
        expect(screen.queryByText('Obsolete resource')).toBeNull();
        expect(view.container.querySelector('.ant-alert-error')).toBeNull();
        expect(onUnavailable).not.toHaveBeenCalled();
      },
    );

    it('offers retry without empty-list copy when the workspace resource read fails', async () => {
      server.use(mockApi.network('get', domain === 'storages' ? '/storages' : '/notifiers'));
      const view = renderScreen(<ResourceScreen workspace={workspace} />, { isStrictMode: true });
      const retry = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.queryByText(emptyText)).toBeNull();

      server.use(
        domain === 'storages'
          ? mockApi.get('/storages', 200, { items: [] })
          : mockApi.get('/notifiers', 200, { items: [] }),
      );

      await view.user.click(retry);

      expect(await screen.findByText(emptyText)).toBeTruthy();
    });

    it('clears the previous workspace and requests app reconciliation after confirmed loss of access', async () => {
      server.use(
        domain === 'storages'
          ? mockApi.get('/storages', 200, { items: [] })
          : mockApi.get('/notifiers', 200, { items: [] }),
      );

      const onUnavailable = vi.fn();
      const view = renderScreen(
        <ResourceScreen workspace={workspace} onUnavailable={onUnavailable} />,
      );
      await screen.findByText(emptyText);

      server.use(
        domain === 'storages'
          ? mockApi.get('/storages', 404, buildProblem('not_found', 404))
          : mockApi.get('/notifiers', 404, buildProblem('not_found', 404)),
      );

      view.rerender(
        <ResourceScreen
          workspace={{ ...workspace, id: 'workspace-b', name: 'Team B' }}
          onUnavailable={onUnavailable}
        />,
      );

      await waitFor(() => expect(onUnavailable).toHaveBeenCalledWith('workspace-b'));
      expect(screen.queryByText(emptyText)).toBeNull();
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy();
    });
  });
}
