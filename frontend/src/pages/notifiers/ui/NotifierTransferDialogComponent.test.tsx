// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import type { Notifier } from '../../../entities/notifiers';
import type { Workspace } from '../../../entities/workspaces';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { inspectApiRequest } from '../../../shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { NotifierTransferDialogComponent } from './NotifierTransferDialogComponent';

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

    fireEvent.mouseDown(await screen.findByRole('combobox', { name: 'Target workspace' }));
    fireEvent.click(
      await screen.findByText(targetWorkspace.name, {
        selector: '.ant-select-item-option-content',
      }),
    );
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
      screen.getByText(targetWorkspace.name, { selector: '.ant-select-selection-item' }),
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
