// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
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
  const addActionName = domain === 'storages' ? 'Add storage' : 'Add notifier';
  const listPath = domain === 'storages' ? '/storages' : '/notifiers';
  const buildResource = (name: string) => ({
    id: `${name}-resource`,
    workspaceId: workspace.id,
    name: `Archive ${name}`,
    settings: domain === 'storages' ? { type: 'LOCAL' } : { type: 'DISCORD' },
    ...(domain === 'notifiers' ? { lastSendError: null } : {}),
  });
  const mockSelectableResources = (resources: ReturnType<typeof buildResource>[]) => [
    http.get(`*${API_BASE_PATH}/${domain}`, () => HttpResponse.json({ items: resources })),
    http.get(`*${API_BASE_PATH}/${domain}/:resourceId`, ({ params }) =>
      HttpResponse.json(resources.find((resource) => resource.id === params.resourceId)),
    ),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/databases', 200, { items: [] }),
  ];
  const submitTransferTo = async (
    targetWorkspaceName: string,
    user: ReturnType<typeof renderScreen>['user'],
  ) => {
    await user.click(
      await screen.findByRole('button', {
        name: domain === 'storages' ? 'Transfer storage to another workspace' : 'Transfer',
      }),
    );
    const targetSelector = await screen.findByRole('combobox', { name: 'Target workspace' });

    fireEvent.mouseDown(targetSelector);
    const targetOption = await screen.findByText(targetWorkspaceName, {
      selector: '.ant-select-item-option-content',
    });

    fireEvent.click(targetOption);
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Transfer' }));
  };

  describe(`${domain} workspace ownership`, () => {
    it.each(
      (['delete', 'transfer'] as const).flatMap((operation) =>
        [false, true].map((isLastResource) => ({ operation, isLastResource })),
      ),
    )(
      'reconciles selection and preference when $operation succeeds with last resource=$isLastResource',
      async ({ operation, isLastResource }) => {
        const resources = ['first', 'second'].map(buildResource);
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
          await submitTransferTo('Target workspace', view.user);
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
          await screen.findByRole('button', { name: addActionName });

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

    it('sends the chosen workspace as the transfer target when several workspaces are offered', async () => {
      const transferredResource = buildResource('first');
      const sentTransfers: { resourceId: unknown; body: unknown }[] = [];
      server.use(
        ...mockSelectableResources([transferredResource]),
        mockApi.get('/workspaces', 200, {
          items: [
            workspace,
            { ...workspace, id: 'skipped-target-id', name: 'Skipped workspace' },
            { ...workspace, id: 'chosen-target-id', name: 'Chosen workspace' },
          ],
        }),
        http.post(
          `*${API_BASE_PATH}/${domain}/:resourceId/transfer`,
          async ({ request, params }) => {
            sentTransfers.push({ resourceId: params.resourceId, body: await request.json() });

            return new HttpResponse(null, { status: 204 });
          },
        ),
      );
      const view = renderScreen(<ResourceScreen workspace={workspace} />);

      await view.user.click(await screen.findByRole('button', { name: transferredResource.name }));
      await submitTransferTo('Chosen workspace', view.user);

      await waitFor(() => expect(sentTransfers).toHaveLength(1));
      expect(sentTransfers[0]).toEqual({
        resourceId: transferredResource.id,
        body: { targetWorkspaceId: 'chosen-target-id' },
      });
    });

    it('keeps the dialog, the resource and its selection when a transfer is refused', async () => {
      const refusedResource = buildResource('first');
      server.use(
        ...mockSelectableResources([refusedResource]),
        mockApi.get('/workspaces', 200, {
          items: [workspace, { ...workspace, id: 'target-id', name: 'Target workspace' }],
        }),
        http.post(`*${API_BASE_PATH}/${domain}/:resourceId/transfer`, () =>
          HttpResponse.json(buildProblem('in_use', 409), { status: 409 }),
        ),
      );
      const selectionKey = `${domain === 'storages' ? 'selectedStorageId' : 'selectedNotifierId'}_${workspace.id}`;
      const onUnavailable = vi.fn();
      const view = renderScreen(
        <ResourceScreen workspace={workspace} onUnavailable={onUnavailable} />,
      );

      await view.user.click(await screen.findByRole('button', { name: refusedResource.name }));
      await submitTransferTo('Target workspace', view.user);

      const dialog = within(screen.getByRole('dialog'));

      expect(
        await dialog.findByText(
          'It is still in use by a database or by a running backup or restore. Try again once nothing uses it.',
        ),
      ).toBeTruthy();
      expect(
        screen.getByRole('button', { name: refusedResource.name }).getAttribute('aria-pressed'),
      ).toBe('true');
      expect(localStorage.getItem(selectionKey)).toBe(refusedResource.id);
      expect(screen.getByRole('button', { name: addActionName })).toBeTruthy();
      expect(onUnavailable).not.toHaveBeenCalled();
    });

    it.each([
      {
        scenario: 'an unreadable body with a success status',
        status: 200,
        reason: 'The server could not complete the request (HTTP 200). Please try again later.',
      },
      {
        scenario: 'a proxy page with a gateway status',
        status: 502,
        reason: 'The server could not complete the request (HTTP 502). Please try again later.',
      },
      {
        scenario: 'JSON of another shape with a success status',
        status: 200,
        body: '{"unexpected":"private proxy page"}',
        reason: 'Something went wrong. Please try again.',
      },
    ])(
      'shows safe translated feedback with a retry when the list answers with $scenario',
      async ({ status, body = '<html><body>private proxy page</body></html>', reason }) => {
        server.use(mockApi.malformed({ method: 'get', path: listPath, status, body }));
        const view = renderScreen(<ResourceScreen workspace={workspace} />);

        expect(await screen.findByText(reason)).toBeTruthy();
        expect(document.body.textContent).not.toContain('private proxy page');
        expect(screen.queryByRole('button', { name: addActionName })).toBeNull();

        server.use(
          domain === 'storages'
            ? mockApi.get('/storages', 200, { items: [] })
            : mockApi.get('/notifiers', 200, { items: [] }),
        );

        await view.user.click(screen.getByRole('button', { name: 'Retry' }));

        expect(await screen.findByRole('button', { name: addActionName })).toBeTruthy();
        expect(screen.queryByText(reason)).toBeNull();
      },
    );

    it('shows the generic message without the server text when the list answers with an unknown code', async () => {
      server.use(
        http.get(`*${API_BASE_PATH}/${domain}`, () =>
          HttpResponse.json(
            {
              type: 'urn:databasus:problem:future_code',
              title: 'private server title',
              status: 409,
              detail: 'private server detail',
              code: 'future_code',
            },
            { status: 409 },
          ),
        ),
      );
      renderScreen(<ResourceScreen workspace={workspace} />);

      expect(await screen.findByText('Something went wrong. Please try again.')).toBeTruthy();
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy();
      expect(document.body.textContent).not.toContain('private server');
      expect(document.body.textContent).not.toContain('future_code');
    });

    it('offers retry before showing the add action when the workspace resource read fails', async () => {
      server.use(mockApi.network('get', domain === 'storages' ? '/storages' : '/notifiers'));
      const view = renderScreen(<ResourceScreen workspace={workspace} />, { isStrictMode: true });
      const retry = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.queryByRole('button', { name: addActionName })).toBeNull();

      server.use(
        domain === 'storages'
          ? mockApi.get('/storages', 200, { items: [] })
          : mockApi.get('/notifiers', 200, { items: [] }),
      );

      await view.user.click(retry);

      expect(await screen.findByRole('button', { name: addActionName })).toBeTruthy();
    });

    it('clears the loaded list, requests app reconciliation and offers retry when a refresh confirms loss of access', async () => {
      const resource = buildResource('first');
      const detailRead = createPendingApiResponse();
      server.use(
        http.get(`*${API_BASE_PATH}/${domain}/:resourceId`, () => detailRead.response),
        ...mockSelectableResources([resource]),
      );

      const onUnavailable = vi.fn();
      renderScreen(<ResourceScreen workspace={workspace} onUnavailable={onUnavailable} />);
      await screen.findByRole('button', { name: resource.name });

      server.use(
        domain === 'storages'
          ? mockApi.get('/storages', 404, buildProblem('not_found', 404))
          : mockApi.get('/notifiers', 404, buildProblem('not_found', 404)),
      );
      detailRead.respond(HttpResponse.json(buildProblem('not_found', 404), { status: 404 }));

      await waitFor(() => expect(onUnavailable).toHaveBeenCalledWith(workspace.id));
      await waitFor(() => expect(screen.queryByRole('button', { name: resource.name })).toBeNull());
      expect(screen.queryByRole('button', { name: addActionName })).toBeNull();
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy();
    });
  });
}
