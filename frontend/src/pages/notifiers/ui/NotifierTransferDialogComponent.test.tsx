// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { databaseApi } from '@/entities/databases';
import type { Notifier } from '@/entities/notifiers';
import type { Workspace } from '@/entities/workspaces';
import { NotifierTransferDialogComponent } from '@/pages/notifiers/ui/NotifierTransferDialogComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { inspectApiRequest } from '@/shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const sourceWorkspace: Workspace = {
  id: 'source-workspace-id',
  name: 'Production',
  createdAt: '2026-01-01T00:00:00Z',
  currentUserRole: 'WORKSPACE_ADMIN',
};

const targetWorkspace: Workspace = {
  ...sourceWorkspace,
  id: 'target-workspace-id',
  name: 'Staging',
};

const transferredNotifier: Notifier = {
  id: 'notifier-id',
  workspaceId: sourceWorkspace.id,
  name: 'Team alerts',
  lastSendError: null,
  settings: { type: 'DISCORD' },
};

const databaseOfNotifier: Database = {
  id: 'database-id',
  workspaceId: sourceWorkspace.id,
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
  notifierIds: [transferredNotifier.id],
};

const mockNotifierDatabases = (
  notifierDatabases: Database[],
  databaseListQueries: URLSearchParams[],
) =>
  http.get(`*${API_BASE_PATH}/databases`, ({ request }) => {
    databaseListQueries.push(new URL(request.url).searchParams);
    const page: ApiComponents['schemas']['DatabasePage'] = { items: notifierDatabases };
    return HttpResponse.json(page);
  });

describe('NotifierTransferDialogComponent', () => {
  it('shows the block and offers no transfer when a database uses the notifier', async () => {
    const databaseListQueries: URLSearchParams[] = [];
    mockServer.use(mockNotifierDatabases([databaseOfNotifier], databaseListQueries));
    const onClose = vi.fn();
    const view = renderScreen(
      <NotifierTransferDialogComponent
        notifier={transferredNotifier}
        onClose={onClose}
        onTransferred={() => {}}
      />,
    );

    expect(
      await screen.findByText(
        'This notifier is used by some databases. Please transfer or remove related databases first.',
      ),
    ).toBeTruthy();
    expect(databaseListQueries).toHaveLength(1);
    expect(databaseListQueries[0].get('notifierId')).toBe(transferredNotifier.id);
    expect(databaseListQueries[0].get('workspaceId')).toBe(sourceWorkspace.id);
    expect(screen.queryByRole('combobox', { name: 'Target workspace' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Transfer' })).toBeNull();

    await view.user.click(screen.getByRole('button', { name: 'OK' }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('reloads the dialog data through Retry when the data fails to load', async () => {
    mockServer.use(mockApi.get('/databases', 500, buildProblem('internal', 500)));
    const view = renderScreen(
      <NotifierTransferDialogComponent
        notifier={transferredNotifier}
        onClose={() => {}}
        onTransferred={() => {}}
      />,
    );

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    mockServer.use(
      mockNotifierDatabases([], []),
      mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
    );

    await view.user.click(retryButton);

    expect(await screen.findByRole('combobox', { name: 'Target workspace' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('keeps the dialog and the chosen workspace and shows the translated reason when the transfer is refused', async () => {
    mockServer.use(
      mockNotifierDatabases([], []),
      mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
      mockApi.post('/notifiers/{notifierId}/transfer', 409, buildProblem('in_use', 409)),
    );
    const onClose = vi.fn();
    const onTransferred = vi.fn();
    const view = renderScreen(
      <NotifierTransferDialogComponent
        notifier={transferredNotifier}
        onClose={onClose}
        onTransferred={onTransferred}
      />,
    );

    fireEvent.click(await screen.findByRole('combobox', { name: 'Target workspace' }));
    fireEvent.click(await screen.findByRole('option', { name: targetWorkspace.name }));
    await view.user.click(screen.getByRole('button', { name: /Transfer/ }));

    expect(
      await screen.findByText(
        'It is still in use by a database or by a running backup or restore. Try again once nothing uses it.',
      ),
    ).toBeTruthy();
    expect(screen.getByRole('dialog')).toBeTruthy();
    expect(screen.queryByText('in_use')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(
      within(screen.getByRole('combobox', { name: 'Target workspace' })).getByText(
        targetWorkspace.name,
      ),
    ).toBeTruthy();
    expect(onTransferred).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();

    mockServer.use(
      http.post(
        `*${API_BASE_PATH}/notifiers/:notifierId/transfer`,
        () => new HttpResponse(null, { status: 204 }),
      ),
    );

    await view.user.click(screen.getByRole('button', { name: /Transfer/ }));

    await waitFor(() => expect(onTransferred).toHaveBeenCalledTimes(1));

    const transfers = mockServer.getRequests().filter((request) => request.method === 'POST');

    expect(transfers).toHaveLength(2);

    for (const transfer of transfers) {
      const inspectedTransfer = await inspectApiRequest(transfer, {
        method: 'post',
        path: '/notifiers/{notifierId}/transfer',
      });

      expect(inspectedTransfer.params.notifierId).toBe(transferredNotifier.id);
      expect(inspectedTransfer.body).toEqual({ targetWorkspaceId: targetWorkspace.id });
    }
  });
});

const TransferDialog = ({
  resourceId = transferredNotifier.id,
  onClose,
  onTransferred,
}: {
  resourceId?: string;
  onClose: () => void;
  onTransferred: () => void;
}) => (
  <NotifierTransferDialogComponent
    notifier={{ ...transferredNotifier, id: resourceId }}
    onClose={onClose}
    onTransferred={onTransferred}
  />
);
const usagePath = '/databases';
const availableTargets = () =>
  mockServer.use(
    mockApi.get(usagePath, 200, { items: [] }),
    mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
  );
const chooseTarget = async () => {
  fireEvent.click(await screen.findByRole('combobox', { name: 'Target workspace' }));

  expect(screen.queryByRole('option', { name: sourceWorkspace.name })).toBeNull();

  fireEvent.click(await screen.findByRole('option', { name: targetWorkspace.name }));
};

describe('Notifier transfer lifecycle', () => {
  it('loads current targets when Strict Mode cancels its first setup', async () => {
    const usageRead = vi.spyOn(databaseApi, 'listAllDatabases');
    availableTargets();
    renderScreen(<TransferDialog onClose={() => {}} onTransferred={() => {}} />, {
      isStrictMode: true,
    });
    await screen.findByRole('combobox', { name: 'Target workspace' });
    expect(usageRead.mock.calls.length).toBeGreaterThanOrEqual(2);
    expect(usageRead.mock.calls[0][1]?.signal?.aborted).toBe(true);
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('submits once and calls the current completion handler when transfer options change while pending', async () => {
    availableTargets();
    const transfer = createPendingApiResponse();
    let transfers = 0;
    mockServer.use(
      http.post(`*${API_BASE_PATH}/notifiers/:resourceId/transfer`, () => {
        transfers++;
        return transfer.response;
      }),
    );
    const oldTransferred = vi.fn();
    const currentTransferred = vi.fn();
    const view = renderScreen(<TransferDialog onClose={() => {}} onTransferred={oldTransferred} />);

    await chooseTarget();
    const submit = screen.getByRole('button', { name: 'Transfer' });
    act(() => {
      submit.click();
      submit.click();
    });
    await waitFor(() => expect(transfers).toBe(1));

    view.rerender(<TransferDialog onClose={() => {}} onTransferred={currentTransferred} />);
    transfer.respond(new HttpResponse(null, { status: 204 }));
    await waitFor(() => expect(currentTransferred).toHaveBeenCalledTimes(1));

    expect(oldTransferred).not.toHaveBeenCalled();
    expect(transfers).toBe(1);
  });

  it.each([204, 409])(
    'resets the target and ignores an obsolete HTTP %i transfer after the resource changes',
    async (status) => {
      availableTargets();
      const transfer = createPendingApiResponse();
      let transfers = 0;
      mockServer.use(
        http.post(`*${API_BASE_PATH}/notifiers/:resourceId/transfer`, () => {
          transfers++;
          return transfer.response;
        }),
      );
      const onTransferred = vi.fn();
      const view = renderScreen(
        <TransferDialog onClose={() => {}} onTransferred={onTransferred} />,
      );

      await chooseTarget();
      await view.user.click(screen.getByRole('button', { name: 'Transfer' }));
      await waitFor(() => expect(transfers).toBe(1));

      view.rerender(
        <TransferDialog
          resourceId="replacement-id"
          onClose={() => {}}
          onTransferred={onTransferred}
        />,
      );
      await screen.findByRole('combobox', { name: 'Target workspace' });

      expect(
        within(screen.getByRole('combobox', { name: 'Target workspace' })).queryByText(
          targetWorkspace.name,
        ),
      ).toBeNull();
      expect((screen.getByRole('button', { name: 'Transfer' }) as HTMLButtonElement).disabled).toBe(
        true,
      );

      await finishApiResponse(
        mockServer,
        `/api/v1/notifiers/${transferredNotifier.id}/transfer`,
        () =>
          transfer.respond(
            status === 204
              ? new HttpResponse(null, { status: 204 })
              : HttpResponse.json(buildProblem('in_use', 409), { status: 409 }),
          ),
      );

      expect(onTransferred).not.toHaveBeenCalled();
      expect(screen.queryByRole('alert')).toBeNull();

      await chooseTarget();

      expect((screen.getByRole('button', { name: 'Transfer' }) as HTMLButtonElement).disabled).toBe(
        false,
      );
    },
  );

  it.each([200, 500])(
    'stops the target read and ignores HTTP %i usage results after closing',
    async (status) => {
      const usageRead = mockApi.pending('get', usagePath);
      mockServer.use(usageRead.handler);
      const onTransferred = vi.fn();
      const onClose = vi.fn(() => view.rerender(<></>));
      const view = renderScreen(<TransferDialog onClose={onClose} onTransferred={onTransferred} />);
      await waitFor(() => expect(mockServer.getRequests()).toHaveLength(1));
      await view.user.click(screen.getByRole('button', { name: 'Close' }));
      await finishCancelledApiResponse(mockServer, `/api/v1${usagePath}`, () =>
        usageRead.respond(
          status === 200
            ? HttpResponse.json({ items: [] })
            : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
        ),
      );

      expect(screen.queryByRole('dialog')).toBeNull();
      expect(onClose).toHaveBeenCalledTimes(1);
      expect(onTransferred).not.toHaveBeenCalled();
      expect(mockServer.getRequests()).toHaveLength(1);
    },
  );

  it.each([204, 409])(
    'ignores a pending HTTP %i transfer after closing the dialog',
    async (status) => {
      availableTargets();
      const transfer = createPendingApiResponse();
      mockServer.use(
        http.post(`*${API_BASE_PATH}/notifiers/:resourceId/transfer`, () => transfer.response),
      );
      const onTransferred = vi.fn();
      const onClose = vi.fn(() => view.rerender(<></>));
      const view = renderScreen(<TransferDialog onClose={onClose} onTransferred={onTransferred} />);
      await chooseTarget();
      await view.user.click(screen.getByRole('button', { name: 'Transfer' }));
      await waitFor(() =>
        expect(
          mockServer.getRequests().filter((request) => request.method === 'POST'),
        ).toHaveLength(1),
      );
      await view.user.click(screen.getByRole('button', { name: 'Cancel' }));
      await finishApiResponse(
        mockServer,
        `/api/v1/notifiers/${transferredNotifier.id}/transfer`,
        () =>
          transfer.respond(
            status === 204
              ? new HttpResponse(null, { status: 204 })
              : HttpResponse.json(buildProblem('in_use', 409), { status: 409 }),
          ),
      );

      expect(screen.queryByRole('dialog')).toBeNull();
      expect(onTransferred).not.toHaveBeenCalled();
      expect(onClose).toHaveBeenCalledTimes(1);
    },
  );
});
