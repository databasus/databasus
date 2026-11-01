// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import {
  StorageSwitchHistoryComponent,
  type StorageSwitch,
} from '../../../../entities/backups/config';
import { API_BASE_PATH } from '../../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const operation: StorageSwitch = {
  id: 'switch-id',
  databaseId: 'deleted-database',
  workspaceId: 'workspace-id',
  previousStorageId: 'old-storage',
  targetStorageId: 'new-storage',
  status: 'CLEANUP_PENDING',
  createdAt: '2026-10-30T10:00:00Z',
  cleanupDeadline: '2026-10-31T10:00:00Z',
  total: 3,
  pending: 3,
  deleted: 0,
  remaining: 0,
};

const completed: StorageSwitch = {
  ...operation,
  status: 'COMPLETED_WITH_REMAINDERS',
  pending: 0,
  deleted: 2,
  remaining: 1,
  completedAt: '2026-10-30T12:00:00Z',
};

describe('StorageSwitchHistoryComponent', () => {
  it('discovers a pending switch after reload and polls until the remaining files are reported', async () => {
    let statusReads = 0;
    const scheduled = vi.spyOn(globalThis, 'setTimeout');

    mockServer.use(
      http.get(`*${API_BASE_PATH}/storage-switches`, ({ request }) => {
        const query = new URL(request.url).searchParams;
        expect(query.get('workspaceId')).toBe(operation.workspaceId);
        expect(query.get('databaseId')).toBe(operation.databaseId);
        statusReads++;
        return HttpResponse.json({ items: [statusReads === 1 ? operation : completed] });
      }),
    );
    mockServer.use(
      http.get(`*${API_BASE_PATH}/storage-switches/:storageSwitchId`, () => {
        statusReads++;
        return HttpResponse.json(completed);
      }),
    );

    const view = renderScreen(
      <StorageSwitchHistoryComponent
        workspaceId={operation.workspaceId}
        databaseId={operation.databaseId}
      />,
    );

    expect(
      await screen.findByText('Storage switched. Old files are being cleaned up.'),
    ).toBeTruthy();
    expect(screen.getByText('Storage after switch: new-storage')).toBeTruthy();
    expect(scheduled.mock.calls.some((call) => call[1] === 2000)).toBe(true);
    expect(
      await screen.findByText(
        'Storage switched successfully, but deletion of some old files could not be confirmed. Remove these files manually from the old storage.',
        {},
      ),
    ).toBeTruthy();
    expect(screen.getByText('Files: 3. Pending: 0. Deleted: 2. Remaining: 1.')).toBeTruthy();
    expect(statusReads).toBe(2);

    view.unmount();
    scheduled.mockRestore();
  });

  it('keeps the last accepted state and retries only the status when a read fails', async () => {
    let statusReads = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/storage-switches`, () => {
        statusReads++;
        return statusReads === 1
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json({ items: [completed] });
      }),
    );
    mockServer.use(
      http.get(`*${API_BASE_PATH}/storage-switches/:storageSwitchId`, () => {
        statusReads++;
        return HttpResponse.json(completed);
      }),
    );

    renderScreen(
      <StorageSwitchHistoryComponent
        workspaceId={operation.workspaceId}
        initialOperation={operation}
      />,
    );

    fireEvent.click(await screen.findByRole('button', { name: 'Retry status' }));
    expect(screen.getByText('Files: 3. Pending: 3. Deleted: 0. Remaining: 0.')).toBeTruthy();
    expect(await screen.findByText('Files: 3. Pending: 0. Deleted: 2. Remaining: 1.')).toBeTruthy();
    expect(statusReads).toBe(2);
  });

  it('shows workspace history for a deleted database and loads older operations by cursor', async () => {
    const queries: URLSearchParams[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/storage-switches`, ({ request }) => {
        const query = new URL(request.url).searchParams;
        queries.push(query);
        return HttpResponse.json(
          query.has('after')
            ? {
                items: [
                  { ...completed, id: 'older-switch', databaseId: 'another-deleted-database' },
                ],
              }
            : { items: [completed], nextCursor: 'older-page' },
        );
      }),
    );

    renderScreen(<StorageSwitchHistoryComponent workspaceId={operation.workspaceId} />);

    expect(await screen.findByText('Database: deleted-database')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Show more' }));
    expect(await screen.findByText('Database: another-deleted-database')).toBeTruthy();
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Show more' })).toBeNull());
    expect(queries[0].has('databaseId')).toBe(false);
    expect(queries[1].get('after')).toBe('older-page');
  });

  it('stops polling when the screen unmounts with cleanup still pending', async () => {
    const scheduled = vi.spyOn(globalThis, 'setTimeout');
    const cleared = vi.spyOn(globalThis, 'clearTimeout');
    mockServer.use(mockApi.get('/storage-switches', 200, { items: [operation] }));

    const view = renderScreen(
      <StorageSwitchHistoryComponent workspaceId={operation.workspaceId} />,
    );
    await screen.findByText('Storage switched. Old files are being cleaned up.');

    const pollIndex = scheduled.mock.calls.findIndex((call) => call[1] === 2000);
    expect(pollIndex).toBeGreaterThanOrEqual(0);
    const pollTimer = scheduled.mock.results[pollIndex].value;

    view.unmount();
    expect(cleared).toHaveBeenCalledWith(pollTimer);
    scheduled.mockRestore();
    cleared.mockRestore();
  });
});
