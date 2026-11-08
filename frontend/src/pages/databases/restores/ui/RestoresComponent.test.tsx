// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { type Restore, restoreApi } from '@/entities/restores';
import type { LogicalBackup } from '@/pages/databases/logical-backups/model/logicalBackupContract';
import { RestoresComponent } from '@/pages/databases/restores/ui/RestoresComponent';
import { API_BASE_PATH } from '@/shared/api';
import { advanceClockUntil } from '@/shared/lib/test/advanceClockUntil';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { renderScreen } from '@/shared/lib/test/renderScreen';

dayjs.extend(utc);

const mockServer = setUpMockApi();

const buildDatabase = (type: DatabaseType): Database => ({
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(type),
  notifierIds: [],
});

const completedBackup: LogicalBackup = {
  id: 'backup-id',
  databaseId: 'database-id',
  storageId: 'storage-id',
  status: 'COMPLETED',
  sizeMb: 12,
  rawDatabaseSizeMb: 40,
  durationMs: 3_000,
  encryption: 'NONE',
  verificationStatus: 'NOT_VERIFIED',
  createdAt: '2026-10-27T10:00:00Z',
};

describe('RestoresComponent', () => {
  it('offers no target form for a physical database', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [] }));
    renderScreen(
      <RestoresComponent
        database={buildDatabase(DatabaseType.POSTGRES_PHYSICAL)}
        backup={completedBackup}
      />,
    );

    expect(await screen.findByText('No restores yet')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Select database to restore to' })).toBeNull();
  });
});

describe('RestoresComponent actions', () => {
  const mysqlDatabase = buildDatabase(DatabaseType.MYSQL);

  const buildRestore = (status: Restore['status']) =>
    ({
      id: 'restore-id',
      backupId: completedBackup.id,
      status,
      durationMs: 0,
      createdAt: '2026-10-27T10:00:00Z',
    }) as Restore;

  const SELECT_TARGET_NAME = /Select database to restore to$/;

  it('keeps the target selection disabled and closed with a running restore', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [buildRestore('IN_PROGRESS')] }));
    renderScreen(<RestoresComponent database={mysqlDatabase} backup={completedBackup} />);

    const selectTargetButton = await screen.findByRole<HTMLButtonElement>('button', {
      name: SELECT_TARGET_NAME,
    });

    expect(selectTargetButton.disabled).toBe(true);

    fireEvent.click(selectTargetButton);

    expect(screen.queryByPlaceholderText(/host$/)).toBeNull();
  });

  it.each([
    { status: 'COMPLETED', label: 'Successful' },
    { status: 'FAILED', label: 'Failed' },
    { status: 'CANCELED', label: 'Canceled' },
  ] as const)(
    'offers no cancel and an enabled target selection with a $status restore',
    async ({ status, label }) => {
      mockServer.use(mockApi.get('/restores', 200, { items: [buildRestore(status)] }));
      renderScreen(<RestoresComponent database={mysqlDatabase} backup={completedBackup} />);

      expect(await screen.findByText(label)).toBeTruthy();

      expect(screen.queryByRole('button', { name: 'Cancel restore' })).toBeNull();
      expect(
        screen.getByRole<HTMLButtonElement>('button', { name: SELECT_TARGET_NAME }).disabled,
      ).toBe(false);
    },
  );
});

describe('RestoresComponent failures', () => {
  const mysqlDatabase = buildDatabase(DatabaseType.MYSQL);

  const inProgressRestore = {
    id: 'restore-id',
    backupId: completedBackup.id,
    status: 'IN_PROGRESS',
    durationMs: 0,
    createdAt: '2026-10-27T10:00:00Z',
  } as Restore;

  const INTERNAL_ERROR = 'An internal server error occurred. Please try again later.';

  const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });

  it('shows no alert and keeps the restores when polls fail', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const listCall = vi.spyOn(restoreApi, 'listAllRestores');
    let listRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/restores`, () =>
        ++listRequests === 1 ? HttpResponse.json({ items: [inProgressRestore] }) : failedResponse(),
      ),
    );
    vi.useFakeTimers();
    renderScreen(<RestoresComponent database={mysqlDatabase} backup={completedBackup} />);
    await waitForCondition(() => screen.getByText('In progress'));

    await advanceClockUntil(() => expect(listCall.mock.results.length).toBeGreaterThanOrEqual(3));
    await act(async () => {
      await Promise.allSettled(listCall.mock.results.map((call) => call.value));
    });

    expect(showAlert).not.toHaveBeenCalled();
    expect(screen.queryByText(INTERNAL_ERROR)).toBeNull();
    expect(document.querySelector('[role="alert"]')).toBeNull();
    expect(screen.getByText('In progress')).toBeTruthy();
  });

  it('shows a retry and no target selection when the first load fails', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    mockServer.use(http.get(`*${API_BASE_PATH}/restores`, failedResponse));
    renderScreen(<RestoresComponent database={mysqlDatabase} backup={completedBackup} />);

    expect(await screen.findByRole('button', { name: 'Retry' })).toBeTruthy();
    expect(
      screen.getByText('An internal server error occurred. Please try again later.'),
    ).toBeTruthy();
    expect(screen.queryByText('No restores yet')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Select database to restore to' })).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('sends no refresh after a failed first load, then shows restores and refreshes after a retry', async () => {
    const listCall = vi.spyOn(restoreApi, 'listAllRestores');
    let isListFailing = true;
    let listRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/restores`, () => {
        listRequests += 1;

        return isListFailing ? failedResponse() : HttpResponse.json({ items: [inProgressRestore] });
      }),
    );
    vi.useFakeTimers();
    renderScreen(<RestoresComponent database={mysqlDatabase} backup={completedBackup} />);
    await waitForCondition(() => screen.getByRole('button', { name: 'Retry' }));

    await advanceFakeClock(3_000);

    expect(listCall).toHaveBeenCalledOnce();

    isListFailing = false;
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    await waitForCondition(() => screen.getByText('In progress'));

    expect(screen.getByRole('button', { name: /Select database to restore to$/ })).toBeTruthy();

    await advanceClockUntil(() => expect(listRequests).toBeGreaterThan(2));
  });

  it('shows every page of the complete listing', async () => {
    mockServer.use(
      http.get(`*${API_BASE_PATH}/restores`, ({ request }) =>
        new URL(request.url).searchParams.get('after')
          ? HttpResponse.json({
              items: [{ ...inProgressRestore, id: 'older-restore', status: 'CANCELED' }],
            })
          : HttpResponse.json({ items: [inProgressRestore], nextCursor: 'next-cursor' }),
      ),
    );
    renderScreen(<RestoresComponent database={mysqlDatabase} backup={completedBackup} />);

    expect(await screen.findByText('In progress')).toBeTruthy();
    expect(await screen.findByText('Canceled')).toBeTruthy();
  });

  it('reports a failed cancel once and keeps the restore', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    mockServer.use(
      mockApi.get('/restores', 200, { items: [inProgressRestore] }),
      mockApi.post('/restores/{restoreId}/cancel', 403, buildProblem('permission_denied', 403)),
    );
    const { user } = renderScreen(
      <RestoresComponent database={mysqlDatabase} backup={completedBackup} />,
    );

    await user.click(await screen.findByRole('button', { name: 'Cancel restore' }));
    await user.click(await screen.findByRole('button', { name: /^Yes, cancel/ }));

    expect(await screen.findAllByText('You do not have permission to do this.')).toHaveLength(1);
    expect(screen.getByText('In progress')).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('stays silent and refreshes again when the refresh after a successful cancel fails', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const listCall = vi.spyOn(restoreApi, 'listAllRestores');
    const heldRefresh = createPendingApiResponse();
    let listRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/restores`, () => {
        listRequests += 1;

        if (listRequests === 1) return HttpResponse.json({ items: [inProgressRestore] });
        if (listRequests === 2) return heldRefresh.response;

        return HttpResponse.json({ items: [{ ...inProgressRestore, status: 'FAILED' }] });
      }),
      http.post(`*${API_BASE_PATH}/restores/:restoreId/cancel`, () =>
        HttpResponse.json({ ...inProgressRestore, status: 'CANCELED' }, { status: 202 }),
      ),
    );
    vi.useFakeTimers();
    renderScreen(<RestoresComponent database={mysqlDatabase} backup={completedBackup} />);

    await waitForCondition(() => screen.getByRole('button', { name: 'Cancel restore' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel restore' }));
    await waitForCondition(() => screen.getByRole('button', { name: /^Yes, cancel/ }));
    fireEvent.click(screen.getByRole('button', { name: /^Yes, cancel/ }));

    await waitForCondition(() => screen.getByText('Canceled'));
    await advanceClockUntil(() => expect(listRequests).toBe(2));

    await act(async () => {
      heldRefresh.respond(failedResponse());
      await Promise.allSettled([listCall.mock.results[1].value]);
    });

    expect(screen.getByText('Canceled')).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();
    expect(screen.queryByText(INTERNAL_ERROR)).toBeNull();

    await advanceClockUntil(() => screen.getByText('Failed'));

    expect(showAlert).not.toHaveBeenCalled();
  });
});
