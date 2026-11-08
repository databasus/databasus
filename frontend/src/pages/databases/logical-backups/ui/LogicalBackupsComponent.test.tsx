import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useRef } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { BackupConfig } from '@/entities/backup-configs';
import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { restoreApi } from '@/entities/restores';
import { logicalBackupApi } from '@/pages/databases/logical-backups/api/logicalBackupApi';
import type {
  LogicalBackup,
  LogicalBackupPage,
} from '@/pages/databases/logical-backups/model/logicalBackupContract';
import { LogicalBackupsComponent } from '@/pages/databases/logical-backups/ui/LogicalBackupsComponent';
import { verificationRunsApi } from '@/pages/databases/verification/api/verificationRunsApi';
import { API_BASE_PATH } from '@/shared/api';
import { advanceClockUntil } from '@/shared/lib/test/advanceClockUntil';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
// @vitest-environment happy-dom
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

// main.tsx registers the plugin for the application; the list renders times through it.
dayjs.extend(utc);

const mockServer = setUpMockApi();

const backedUpDatabase: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
};

const backupConfigOfDatabase: BackupConfig = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: backedUpDatabase.id,
  isBackupsEnabled: true,
  storageId: 'storage-id',
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'HOURLY' },
    retention: {
      policy: 'TIME_PERIOD',
      period: 'WEEK',
      count: 0,
      gfsHours: 0,
      gfsDays: 0,
      gfsWeeks: 0,
      gfsMonths: 0,
      gfsYears: 0,
    },
    isRetryIfFailed: false,
    maxFailedTriesCount: 1,
    notificationTriggers: [],
  },
};

const buildBackup = (id: string, status: LogicalBackup['status']): LogicalBackup => ({
  id,
  databaseId: backedUpDatabase.id,
  storageId: 'storage-id',
  status,
  sizeMb: 12,
  rawDatabaseSizeMb: 40,
  durationMs: 3_000,
  encryption: 'NONE',
  verificationStatus: 'NOT_VERIFIED',
  createdAt: '2026-10-27T10:00:00Z',
});

// Answers each list request with the next page of the queue and leaves the requests after the
// queue without an answer, so the polled refresh cannot replace what the test checks.
const mockBackupPages = (backupPages: LogicalBackupPage[], requestedCursors: (string | null)[]) => {
  const queuedPages = [...backupPages];

  return http.get(`*${API_BASE_PATH}/logical-backups`, ({ request }) => {
    requestedCursors.push(new URL(request.url).searchParams.get('after'));

    const backupPage = queuedPages.shift();
    if (!backupPage) {
      return createPendingApiResponse().response;
    }

    return HttpResponse.json(backupPage);
  });
};

// The database page scrolls the list inside its own container and hands that container in.
function ScrollingDatabasePage({ isCanManageDBs }: { isCanManageDBs: boolean }) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  return (
    <div ref={scrollContainerRef} className="database-scroll-container">
      <LogicalBackupsComponent
        database={backedUpDatabase}
        isCanManageDBs={isCanManageDBs}
        scrollContainerRef={scrollContainerRef}
      />
    </div>
  );
}

const renderBackupList = (isCanManageDBs = true) => {
  const { container } = renderScreen(<ScrollingDatabasePage isCanManageDBs={isCanManageDBs} />);

  return container.querySelector('.database-scroll-container') as Element;
};

describe('LogicalBackupsComponent', () => {
  afterEach(() => vi.restoreAllMocks());

  it('loads the next page by the cursor and stops after the last page when the list is scrolled down', async () => {
    const requestedCursors: (string | null)[] = [];
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get(`*${API_BASE_PATH}/logical-backups`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        return HttpResponse.json<LogicalBackupPage>(
          cursor
            ? { totalUsageMb: 12, items: [buildBackup('oldest-backup', 'FAILED')] }
            : {
                totalUsageMb: 12,
                items: [buildBackup('newest-backup', 'COMPLETED')],
                nextCursor: 'first-page-cursor',
              },
        );
      }),
    );
    const scrollContainer = renderBackupList();
    await screen.findAllByText('Successful');
    expect(screen.getByRole('columnheader', { name: /^Size 12 MB/ })).toBeTruthy();

    fireEvent.scroll(scrollContainer);

    expect(await screen.findAllByText('Failed')).toHaveLength(2);
    expect(await screen.findAllByText('All backups loaded (2 total)')).toHaveLength(2);
    expect(screen.getByRole('columnheader', { name: /^Size 12 MB/ })).toBeTruthy();
    // The list also polls its first page, so polls may fall between these two requests.
    expect(requestedCursors[0]).toBeNull();
    expect(requestedCursors.filter((cursor) => cursor !== null)).toEqual(['first-page-cursor']);
  });

  it('refreshes the first page and keeps the scrolled tail when the poll answers', async () => {
    const requestedCursors: (string | null)[] = [];
    let hasBackupCompleted = false;
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get(`*${API_BASE_PATH}/logical-backups`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        if (cursor)
          return HttpResponse.json<LogicalBackupPage>({
            totalUsageMb: 0,
            items: [buildBackup('oldest-backup', 'FAILED')],
          });

        return HttpResponse.json<LogicalBackupPage>({
          totalUsageMb: hasBackupCompleted ? 12 : 0,
          items: [buildBackup('newest-backup', hasBackupCompleted ? 'COMPLETED' : 'IN_PROGRESS')],
          nextCursor: 'first-page-cursor',
        });
      }),
    );
    vi.useFakeTimers();
    const scrollContainer = renderBackupList();
    await waitForCondition(() => screen.getAllByText('In progress'));
    expect(screen.getByRole('columnheader', { name: /^Size 0 MB/ })).toBeTruthy();

    fireEvent.scroll(scrollContainer);
    await waitForCondition(() => screen.getAllByText('Failed'));
    expect(screen.getByRole('columnheader', { name: /^Size 0 MB/ })).toBeTruthy();

    hasBackupCompleted = true;
    await advanceClockUntil(() => expect(screen.getAllByText('Successful')).toHaveLength(2));

    expect(screen.getByRole('columnheader', { name: /^Size 12 MB/ })).toBeTruthy();
    expect(screen.queryAllByText('In progress')).toHaveLength(0);
    expect(screen.getAllByText('Failed')).toHaveLength(2);
    expect(requestedCursors.filter((cursor) => cursor !== null)).toEqual(['first-page-cursor']);
    expect(requestedCursors[requestedCursors.length - 1]).toBeNull();
  });

  it('shows the started backup at once when a manual backup is started', async () => {
    const startedDatabaseIds: unknown[] = [];
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      mockBackupPages([{ totalUsageMb: 0, items: [] }], []),
      http.post(`*${API_BASE_PATH}/logical-backups`, async ({ request }) => {
        startedDatabaseIds.push(((await request.json()) as { databaseId: string }).databaseId);

        return HttpResponse.json(buildBackup('started-backup', 'IN_PROGRESS'), { status: 202 });
      }),
    );
    renderBackupList();
    await screen.findAllByText('No backups yet');

    fireEvent.click(screen.getByRole('button', { name: /Make backup right now/ }));

    expect(await screen.findAllByText('In progress')).toHaveLength(2);
    expect(startedDatabaseIds).toEqual([backedUpDatabase.id]);
  });

  it('follows the file link with the token when a completed backup is downloaded', async () => {
    const followedLinks: { href: string; download: string }[] = [];
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      followedLinks.push({ href: this.href, download: this.download });
    });

    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      mockBackupPages(
        [{ totalUsageMb: 0, items: [buildBackup('completed-backup', 'COMPLETED')] }],
        [],
      ),
      mockApi.post('/download-tokens', 201, {
        token: 'single-use-token',
        fileName: 'shop-2026-10-27.dump',
        expiresAt: '2026-10-27T10:05:00Z',
      }),
    );
    renderBackupList();
    await screen.findAllByText('Successful');

    fireEvent.click(screen.getAllByRole('button', { name: /^Download backup file/ })[0]);

    await waitFor(() => expect(followedLinks).toHaveLength(1));
    const followedUrl = new URL(followedLinks[0].href);
    expect(followedUrl.pathname).toBe(`${API_BASE_PATH}/logical-backups/completed-backup/file`);
    expect(followedUrl.searchParams.get('token')).toBe('single-use-token');
    expect(followedLinks[0].download).toBe('shop-2026-10-27.dump');
  });
  it.each([
    ['already_running', 'This is already running. Wait for it to finish and try again.'],
    ['backup_not_completed', 'Only a completed backup can be verified.'],
  ] as const)(
    'shows the translated refusal when verification start returns %s',
    async (code, text) => {
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
        mockBackupPages(
          [{ totalUsageMb: 0, items: [buildBackup('completed-backup', 'COMPLETED')] }],
          [],
        ),
        mockApi.post('/verifications', 409, buildProblem(code, 409)),
      );

      renderBackupList();
      await screen.findAllByText('Successful');
      fireEvent.click(screen.getAllByRole('button', { name: /^Verify restore/ })[0]);
      fireEvent.click(await screen.findByRole('button', { name: 'Queue check' }));
      expect(await screen.findByText(text)).toBeTruthy();
    },
  );
});

describe('LogicalBackupsComponent failures', () => {
  afterEach(() => vi.restoreAllMocks());

  const INTERNAL_ERROR = 'An internal server error occurred. Please try again later.';

  const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });

  const waitForTwoSettledRefreshes = async (listCall: {
    mock: { results: { value: unknown }[] };
  }) => {
    await advanceClockUntil(() => expect(listCall.mock.results.length).toBeGreaterThanOrEqual(3));
    await act(async () => {
      await Promise.allSettled(listCall.mock.results.map((call) => call.value));
    });
  };

  it('shows no alert and keeps the rows when polls fail', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const listCall = vi.spyOn(logicalBackupApi, 'listLogicalBackups');
    let listRequests = 0;
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get(`*${API_BASE_PATH}/logical-backups`, () =>
        ++listRequests === 1
          ? HttpResponse.json<LogicalBackupPage>({
              totalUsageMb: 0,
              items: [buildBackup('completed-backup', 'COMPLETED')],
            })
          : failedResponse(),
      ),
    );
    vi.useFakeTimers();
    renderBackupList();
    await waitForCondition(() => screen.getAllByText('Successful'));

    await waitForTwoSettledRefreshes(listCall);

    expect(showAlert).not.toHaveBeenCalled();
    expect(screen.queryByText(INTERNAL_ERROR)).toBeNull();
    expect(screen.getAllByText('Successful')).toHaveLength(2);
  });

  it('shows a retry instead of the empty text and offers no backup start when the first page fails', async () => {
    const makeBackupButton = () =>
      screen.getByRole<HTMLButtonElement>('button', { name: /Make backup right now/ });
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    let isListFailing = true;
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get(`*${API_BASE_PATH}/logical-backups`, () =>
        isListFailing
          ? failedResponse()
          : HttpResponse.json<LogicalBackupPage>({
              totalUsageMb: 0,
              items: [buildBackup('completed-backup', 'COMPLETED')],
            }),
      ),
    );
    renderBackupList();

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
    expect(screen.queryByText('No backups yet')).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();
    expect(makeBackupButton().disabled).toBe(true);

    isListFailing = false;
    fireEvent.click(retryButton);

    expect(await screen.findAllByText('Successful')).toHaveLength(2);
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(makeBackupButton().disabled).toBe(false);
  });

  it('shows the reason and not disabled backups when the config cannot load', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    let isConfigFailing = true;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        isConfigFailing
          ? failedResponse()
          : HttpResponse.json({ ...backupConfigOfDatabase, isBackupsEnabled: false }),
      ),
      mockBackupPages([{ totalUsageMb: 0, items: [] }], []),
    );
    renderBackupList();

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
    expect(screen.queryByText(/Scheduled backups are disabled/)).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();

    isConfigFailing = false;
    fireEvent.click(retryButton);

    expect(await screen.findByText(/Scheduled backups are disabled/)).toBeTruthy();
  });

  describe('with the restore dialog open', () => {
    const listedRestore = {
      id: 'restore-id',
      backupId: 'completed-backup',
      status: 'IN_PROGRESS',
      durationMs: 0,
      createdAt: '2026-10-27T11:00:00Z',
    };

    const openRestoreDialogOfListedBackup = async (listedRestores: unknown[]) => {
      let isBackupListed = true;
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
        http.get(`*${API_BASE_PATH}/logical-backups`, () =>
          HttpResponse.json<LogicalBackupPage>({
            totalUsageMb: 0,
            items: isBackupListed ? [buildBackup('completed-backup', 'COMPLETED')] : [],
          }),
        ),
        http.get(`*${API_BASE_PATH}/restores`, () =>
          isBackupListed
            ? HttpResponse.json({ items: listedRestores })
            : HttpResponse.json(buildProblem('not_found', 404), { status: 404 }),
        ),
      );
      vi.useFakeTimers();
      renderBackupList();
      await waitForCondition(() => screen.getAllByText('Successful'));
      fireEvent.click(screen.getAllByRole('button', { name: 'Restore from backup' })[0]);
      await waitForCondition(() =>
        screen.getByText(listedRestores.length > 0 ? 'In progress' : 'No restores yet'),
      );

      return { dropBackup: () => (isBackupListed = false) };
    };

    it('keeps the same restores when its backup leaves the list and refreshes fail', async () => {
      const restoreListCall = vi.spyOn(restoreApi, 'listAllRestores');
      const { dropBackup } = await openRestoreDialogOfListedBackup([listedRestore]);

      dropBackup();
      await advanceClockUntil(() => screen.getAllByText('No backups yet'));
      const callsBeforeSettle = restoreListCall.mock.results.length;
      await advanceClockUntil(() =>
        expect(restoreListCall.mock.results.length).toBeGreaterThan(callsBeforeSettle),
      );
      await act(async () => {
        await Promise.allSettled(restoreListCall.mock.results.map((call) => call.value));
      });

      expect(screen.getByText('Restore from backup', { selector: 'h2' })).toBeTruthy();
      expect(screen.getByText('In progress')).toBeTruthy();
    });

    it('keeps the values entered in the target form when a poll drops the backup', async () => {
      const { dropBackup } = await openRestoreDialogOfListedBackup([]);
      fireEvent.click(screen.getByRole('button', { name: 'Select database to restore to' }));
      await waitForCondition(() => screen.getByPlaceholderText(/host$/));
      fireEvent.change(screen.getByPlaceholderText(/host$/), {
        target: { value: 'restore-target.example.com' },
      });

      dropBackup();
      await advanceClockUntil(() => screen.getAllByText('No backups yet'));

      expect((screen.getByPlaceholderText(/host$/) as HTMLInputElement).value).toBe(
        'restore-target.example.com',
      );
    });
  });

  it('does not navigate when a verification start answers after the list left', async () => {
    const verificationStart = vi.spyOn(verificationRunsApi, 'startVerification');
    const start = mockApi.delayed({
      method: 'post',
      path: '/verifications',
      status: 202,
      body: {
        id: 'verification-id',
        databaseId: backedUpDatabase.id,
        backupId: 'completed-backup',
        trigger: 'MANUAL',
        status: 'COMPLETED',
        attemptCount: 1,
        createdAt: '2026-10-27T10:00:00Z',
      },
    });
    const onNavigateToVerifications = vi.fn();
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      mockBackupPages(
        [{ totalUsageMb: 0, items: [buildBackup('completed-backup', 'COMPLETED')] }],
        [],
      ),
      start.handler,
    );
    const view = renderScreen(
      <LogicalBackupsComponent
        database={backedUpDatabase}
        isCanManageDBs
        onNavigateToVerifications={onNavigateToVerifications}
      />,
    );
    await screen.findAllByText('Successful');
    fireEvent.click(screen.getAllByRole('button', { name: /^Verify restore/ })[0]);
    fireEvent.click(await screen.findByRole('button', { name: 'Queue check' }));
    await waitFor(() => expect(verificationStart).toHaveBeenCalled());

    view.unmount();
    await releaseAndSettle(verificationStart, start.release);

    expect(onNavigateToVerifications).not.toHaveBeenCalled();
  });

  describe('its one-shot actions', () => {
    const refusal = 'You do not have permission to do this.';

    const renderListedBackup = async (status: LogicalBackup['status']) => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
        mockBackupPages([{ totalUsageMb: 0, items: [buildBackup('listed-backup', status)] }], []),
      );
      renderBackupList();
      await screen.findAllByText(status === 'COMPLETED' ? 'Successful' : 'In progress');

      return showAlert;
    };

    it('shows the new status after a successful cancel', async () => {
      await renderListedBackup('IN_PROGRESS');
      mockServer.use(
        mockApi.post(
          '/logical-backups/{backupId}/cancel',
          202,
          buildBackup('listed-backup', 'CANCELED'),
        ),
      );

      fireEvent.click(screen.getAllByRole('button', { name: 'Cancel backup' })[0]);

      expect(await screen.findAllByText('Canceled')).toHaveLength(2);
    });

    it('removes the row through the reload after a confirmed delete', async () => {
      let listRequests = 0;
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
        http.get(`*${API_BASE_PATH}/logical-backups`, () => {
          listRequests += 1;

          if (listRequests === 1) {
            return HttpResponse.json<LogicalBackupPage>({
              totalUsageMb: 0,
              items: [buildBackup('listed-backup', 'COMPLETED')],
            });
          }

          // A held poll keeps later polls off, so only the reload can change the list.
          return listRequests === 2
            ? createPendingApiResponse().response
            : HttpResponse.json<LogicalBackupPage>({ totalUsageMb: 0, items: [] });
        }),
        http.delete(
          `*${API_BASE_PATH}/logical-backups/:backupId`,
          () => new HttpResponse(null, { status: 204 }),
        ),
      );
      vi.useFakeTimers();
      renderBackupList();
      await waitForCondition(() => screen.getAllByText('Successful'));
      await advanceClockUntil(() => expect(listRequests).toBe(2));

      fireEvent.click(screen.getAllByRole('button', { name: 'Delete backup' })[0]);
      await waitForCondition(() => screen.getByRole('button', { name: 'Delete' }));
      fireEvent.click(screen.getByRole('button', { name: 'Delete' }));

      await waitForCondition(() => screen.getAllByText('No backups yet'));

      expect(screen.queryByText('Successful')).toBeNull();
      expect(listRequests).toBeGreaterThanOrEqual(3);
    });

    it('shows no message and keeps the row when refreshes fail after a successful cancel', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const listCall = vi.spyOn(logicalBackupApi, 'listLogicalBackups');
      let listRequests = 0;
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
        http.get(`*${API_BASE_PATH}/logical-backups`, () =>
          ++listRequests === 1
            ? HttpResponse.json<LogicalBackupPage>({
                totalUsageMb: 0,
                items: [buildBackup('listed-backup', 'IN_PROGRESS')],
              })
            : failedResponse(),
        ),
        http.post(`*${API_BASE_PATH}/logical-backups/:backupId/cancel`, () =>
          HttpResponse.json(buildBackup('listed-backup', 'CANCELED'), { status: 202 }),
        ),
      );
      vi.useFakeTimers();
      renderBackupList();
      await waitForCondition(() => screen.getAllByText('In progress'));

      fireEvent.click(screen.getAllByRole('button', { name: 'Cancel backup' })[0]);
      await waitForCondition(() => screen.getAllByText('Canceled'));
      const callsAfterCancel = listCall.mock.results.length;
      await advanceClockUntil(() =>
        expect(listCall.mock.results.length).toBeGreaterThan(callsAfterCancel + 1),
      );
      await act(async () => {
        await Promise.allSettled(listCall.mock.results.map((call) => call.value));
      });

      expect(screen.getAllByText('Canceled')).toHaveLength(2);
      expect(screen.queryByText(INTERNAL_ERROR)).toBeNull();
      expect(showAlert).not.toHaveBeenCalled();
    });

    it('shows a refused start once without an alert', async () => {
      const showAlert = await renderListedBackup('COMPLETED');
      mockServer.use(mockApi.post('/logical-backups', 403, buildProblem('permission_denied', 403)));

      fireEvent.click(screen.getByRole('button', { name: /Make backup right now/ }));

      expect(await screen.findAllByText(refusal)).toHaveLength(1);
      expect(showAlert).not.toHaveBeenCalled();
    });

    it('shows a refused cancel once without an alert', async () => {
      const showAlert = await renderListedBackup('IN_PROGRESS');
      mockServer.use(
        mockApi.post(
          '/logical-backups/{backupId}/cancel',
          403,
          buildProblem('permission_denied', 403),
        ),
      );

      fireEvent.click(screen.getAllByRole('button', { name: 'Cancel backup' })[0]);

      expect(await screen.findAllByText(refusal)).toHaveLength(1);
      expect(showAlert).not.toHaveBeenCalled();
    });

    it('shows a refused delete once without an alert', async () => {
      const showAlert = await renderListedBackup('COMPLETED');
      mockServer.use(
        mockApi.delete('/logical-backups/{backupId}', 403, buildProblem('permission_denied', 403)),
      );

      fireEvent.click(screen.getAllByRole('button', { name: 'Delete backup' })[0]);
      fireEvent.click(await screen.findByRole('button', { name: 'Delete' }));

      expect(await screen.findAllByText(refusal)).toHaveLength(1);
      expect(showAlert).not.toHaveBeenCalled();
    });

    it('shows a refused download once without an alert', async () => {
      const showAlert = await renderListedBackup('COMPLETED');
      mockServer.use(mockApi.post('/download-tokens', 403, buildProblem('permission_denied', 403)));

      fireEvent.click(screen.getAllByRole('button', { name: /^Download backup file/ })[0]);

      expect(await screen.findAllByText(refusal)).toHaveLength(1);
      expect(showAlert).not.toHaveBeenCalled();
    });

    it('sends one cancel request when cancel is clicked twice while pending', async () => {
      await renderListedBackup('IN_PROGRESS');
      const cancelCall = vi.spyOn(logicalBackupApi, 'cancelLogicalBackup');
      const cancel = mockApi.delayed({
        method: 'post',
        path: '/logical-backups/{backupId}/cancel',
        status: 202,
        body: buildBackup('listed-backup', 'CANCELED'),
      });
      mockServer.use(cancel.handler);
      const cancelButton = screen.getAllByRole('button', { name: 'Cancel backup' })[0];

      fireEvent.click(cancelButton);
      fireEvent.click(cancelButton);
      await waitFor(() => expect(cancelCall).toHaveBeenCalled());
      await releaseAndSettle(cancelCall, cancel.release);

      expect(cancelCall).toHaveBeenCalledOnce();
    });

    it('sends one token request when a download is clicked twice while pending', async () => {
      await renderListedBackup('COMPLETED');
      const tokenCall = vi.spyOn(logicalBackupApi, 'createDownloadToken');
      const token = mockApi.delayed({
        method: 'post',
        path: '/download-tokens',
        status: 201,
        body: {
          token: 'single-use-token',
          fileName: 'shop.dump',
          expiresAt: '2026-10-27T10:05:00Z',
        },
      });
      vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
      mockServer.use(token.handler);
      const downloadButton = screen.getAllByRole('button', { name: /^Download backup file/ })[0];

      fireEvent.click(downloadButton);
      fireEvent.click(downloadButton);
      await waitFor(() => expect(tokenCall).toHaveBeenCalled());
      await releaseAndSettle(tokenCall, token.release);

      expect(tokenCall).toHaveBeenCalledOnce();
    });

    it('requests a new single-use token for each download', async () => {
      await renderListedBackup('COMPLETED');
      const followedLinks: string[] = [];
      vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
        this: HTMLAnchorElement,
      ) {
        followedLinks.push(new URL(this.href).searchParams.get('token') ?? '');
      });
      let tokenRequests = 0;
      mockServer.use(
        http.post(`*${API_BASE_PATH}/download-tokens`, () =>
          HttpResponse.json(
            {
              token: `token-${++tokenRequests}`,
              fileName: 'shop.dump',
              expiresAt: '2026-10-27T10:05:00Z',
            },
            { status: 201 },
          ),
        ),
      );
      const getDownloadButton = () =>
        screen.getAllByRole('button', { name: /^Download backup file/ })[0];

      fireEvent.click(getDownloadButton());
      await waitFor(() => expect(followedLinks).toEqual(['token-1']));
      await waitFor(() => expect(getDownloadButton().getAttribute('aria-busy')).toBe('false'));
      fireEvent.click(getDownloadButton());

      await waitFor(() => expect(followedLinks).toEqual(['token-1', 'token-2']));
    });
  });

  it('shows a failed next page once, keeps the rows and requests it again on a later scroll', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const requestedCursors: (string | null)[] = [];
    let isNextPageFailing = true;
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get(`*${API_BASE_PATH}/logical-backups`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        if (!cursor) {
          return HttpResponse.json<LogicalBackupPage>({
            totalUsageMb: 0,
            items: [buildBackup('newest-backup', 'COMPLETED')],
            nextCursor: 'first-page-cursor',
          });
        }

        return isNextPageFailing
          ? failedResponse()
          : HttpResponse.json<LogicalBackupPage>({
              totalUsageMb: 0,
              items: [buildBackup('oldest-backup', 'FAILED')],
            });
      }),
    );
    const scrollContainer = renderBackupList();
    await screen.findAllByText('Successful');

    fireEvent.scroll(scrollContainer);

    expect(await screen.findAllByText(INTERNAL_ERROR)).toHaveLength(1);
    expect(screen.getAllByText('Successful')).toHaveLength(2);
    expect(showAlert).not.toHaveBeenCalled();

    isNextPageFailing = false;
    fireEvent.scroll(scrollContainer);

    expect(await screen.findAllByText('Failed')).toHaveLength(2);
    expect(requestedCursors.filter((cursor) => cursor !== null)).toEqual([
      'first-page-cursor',
      'first-page-cursor',
    ]);
  });
});

describe('LogicalBackupsComponent row actions', () => {
  afterEach(() => vi.restoreAllMocks());

  const ROW_ACTION_NAMES = {
    cancel: 'Cancel backup',
    delete: 'Delete backup',
    verify: /^Verify restore/,
    restore: 'Restore from backup',
    download: /^Download backup file/,
  };

  type RowAction = keyof typeof ROW_ACTION_NAMES;

  // The list renders each row twice, as a phone card and as a table row.
  const getOfferedRowActions = () =>
    (Object.keys(ROW_ACTION_NAMES) as RowAction[]).filter(
      (action) => screen.queryAllByRole('button', { name: ROW_ACTION_NAMES[action] }).length === 2,
    );

  const declareListedBackups = (listedBackups: LogicalBackup[]) =>
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      mockApi.get('/logical-backups', 200, { totalUsageMb: 0, items: listedBackups }),
    );

  const getStartButton = () =>
    screen.getByRole<HTMLButtonElement>('button', { name: /Make backup right now/ });

  const startedVerification = {
    id: 'verification-id',
    databaseId: backedUpDatabase.id,
    backupId: 'completed-backup',
    trigger: 'MANUAL' as const,
    status: 'PENDING' as const,
    attemptCount: 0,
    createdAt: '2026-10-27T10:00:00Z',
  };

  it.each([
    { status: 'IN_PROGRESS', label: 'In progress', offeredActions: ['cancel'] },
    {
      status: 'COMPLETED',
      label: 'Successful',
      offeredActions: ['delete', 'verify', 'restore', 'download'],
    },
    { status: 'FAILED', label: 'Failed', offeredActions: [] },
    { status: 'CANCELED', label: 'Canceled', offeredActions: [] },
  ] as const)(
    'offers exactly the actions of the status with a $status backup',
    async ({ status, label, offeredActions }) => {
      declareListedBackups([buildBackup('listed-backup', status)]);
      renderBackupList();

      await screen.findAllByText(label);

      expect(getOfferedRowActions()).toEqual(offeredActions);
    },
  );

  // The server lets a viewer start a backup (logical_backup_access_test.go), unlike a physical one.
  it('offers the backup start, restore and download and no cancel, delete or verify with a viewer role', async () => {
    declareListedBackups([
      buildBackup('completed-backup', 'COMPLETED'),
      buildBackup('running-backup', 'IN_PROGRESS'),
    ]);
    renderBackupList(false);

    await screen.findAllByText('In progress');

    expect(getOfferedRowActions()).toEqual(['restore', 'download']);
    expect(getStartButton().disabled).toBe(false);
  });

  it('sends one start request when the backup start is clicked twice while pending', async () => {
    declareListedBackups([buildBackup('completed-backup', 'COMPLETED')]);
    const startCall = vi.spyOn(logicalBackupApi, 'startLogicalBackup');
    const start = mockApi.delayed({
      method: 'post',
      path: '/logical-backups',
      status: 202,
      body: buildBackup('started-backup', 'IN_PROGRESS'),
    });
    mockServer.use(start.handler);
    renderBackupList();
    await screen.findAllByText('Successful');
    const startButton = getStartButton();

    fireEvent.click(startButton);
    fireEvent.click(startButton);
    await waitFor(() => expect(startCall).toHaveBeenCalled());
    await releaseAndSettle(startCall, start.release);

    expect(startCall).toHaveBeenCalledOnce();
  });

  it('sends one delete request when the confirmation and the row action are clicked again while pending', async () => {
    declareListedBackups([buildBackup('completed-backup', 'COMPLETED')]);
    const deleteCall = vi.spyOn(logicalBackupApi, 'deleteLogicalBackup');
    const deletion = mockApi.delayed({
      method: 'delete',
      path: '/logical-backups/{backupId}',
      status: 204,
      body: undefined,
    });
    mockServer.use(deletion.handler);
    renderBackupList();
    await screen.findAllByText('Successful');

    fireEvent.click(screen.getAllByRole('button', { name: ROW_ACTION_NAMES.delete })[0]);
    const confirmButton = await screen.findByRole('button', { name: 'Delete' });
    fireEvent.click(confirmButton);
    fireEvent.click(confirmButton);
    await waitFor(() => expect(deleteCall).toHaveBeenCalled());
    fireEvent.click(screen.getAllByRole('button', { name: /Delete backup/ })[0]);
    await releaseAndSettle(deleteCall, deletion.release);

    expect(deleteCall).toHaveBeenCalledOnce();
  });

  it('sends one verification request when Queue check is clicked twice while pending', async () => {
    declareListedBackups([buildBackup('completed-backup', 'COMPLETED')]);
    const verificationCall = vi.spyOn(verificationRunsApi, 'startVerification');
    const verification = mockApi.delayed({
      method: 'post',
      path: '/verifications',
      status: 202,
      body: startedVerification,
    });
    mockServer.use(verification.handler);
    renderBackupList();
    await screen.findAllByText('Successful');

    fireEvent.click(screen.getAllByRole('button', { name: ROW_ACTION_NAMES.verify })[0]);
    const queueButton = await screen.findByRole('button', { name: 'Queue check' });
    fireEvent.click(queueButton);
    fireEvent.click(queueButton);
    await waitFor(() => expect(verificationCall).toHaveBeenCalled());
    await releaseAndSettle(verificationCall, verification.release);

    expect(verificationCall).toHaveBeenCalledOnce();
  });

  it('sends one verification request after the row verify action is clicked twice and every dialog is confirmed', async () => {
    declareListedBackups([buildBackup('completed-backup', 'COMPLETED')]);
    const verificationCall = vi.spyOn(verificationRunsApi, 'startVerification');
    const verification = mockApi.delayed({
      method: 'post',
      path: '/verifications',
      status: 202,
      body: startedVerification,
    });
    mockServer.use(verification.handler);
    renderBackupList();
    await screen.findAllByText('Successful');
    const verifyButton = screen.getAllByRole('button', { name: ROW_ACTION_NAMES.verify })[0];

    fireEvent.click(verifyButton);
    fireEvent.click(verifyButton);
    const queueButtons = await screen.findAllByRole('button', { name: 'Queue check' });
    queueButtons.forEach((queueButton) => fireEvent.click(queueButton));
    await waitFor(() => expect(verificationCall).toHaveBeenCalled());
    await releaseAndSettle(verificationCall, verification.release);

    expect(verificationCall).toHaveBeenCalledOnce();
  });

  it('sends a second verification request when the row action is used again after a refusal', async () => {
    declareListedBackups([buildBackup('completed-backup', 'COMPLETED')]);
    const verificationCall = vi.spyOn(verificationRunsApi, 'startVerification');
    mockServer.use(mockApi.post('/verifications', 409, buildProblem('already_running', 409)));
    renderBackupList();
    await screen.findAllByText('Successful');
    const verifyButton = screen.getAllByRole('button', { name: ROW_ACTION_NAMES.verify })[0];

    fireEvent.click(verifyButton);
    fireEvent.click(await screen.findByRole('button', { name: 'Queue check' }));
    await screen.findByText('This is already running. Wait for it to finish and try again.');
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Queue check' })).toBeNull());

    fireEvent.click(verifyButton);
    fireEvent.click(await screen.findByRole('button', { name: 'Queue check' }));

    await waitFor(() => expect(verificationCall).toHaveBeenCalledTimes(2));
  });

  it('opens a new verify dialog after the previous one was declined', async () => {
    declareListedBackups([buildBackup('completed-backup', 'COMPLETED')]);
    renderBackupList();
    await screen.findAllByText('Successful');
    const verifyButton = screen.getAllByRole('button', { name: ROW_ACTION_NAMES.verify })[0];

    fireEvent.click(verifyButton);
    const declinedQueueButton = await screen.findByRole('button', { name: 'Queue check' });
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    fireEvent.click(verifyButton);

    await waitFor(() =>
      expect(
        screen
          .getAllByRole('button', { name: 'Queue check' })
          .some((queueButton) => queueButton !== declinedQueueButton),
      ).toBe(true),
    );
  });

  it.each([
    { verificationStatus: 'VERIFIED_SUCCESSFUL', shownBadges: ['Verified'] },
    { verificationStatus: 'VERIFICATION_FAILED', shownBadges: ['Verification failed'] },
    { verificationStatus: 'NOT_VERIFIED', shownBadges: [] },
  ] as const)(
    'shows only the verification badge of the status with a $verificationStatus backup',
    async ({ verificationStatus, shownBadges }) => {
      declareListedBackups([
        { ...buildBackup('completed-backup', 'COMPLETED'), verificationStatus },
      ]);
      renderBackupList();

      await screen.findAllByText('Successful');

      const listedBadges = ['Verified', 'Verification failed'].filter(
        (badge) => screen.queryAllByText(badge).length === 2,
      );
      expect(listedBadges).toEqual(shownBadges);
    },
  );
});

describe('LogicalBackupsComponent usage', () => {
  it.each([
    DatabaseType.POSTGRES_LOGICAL,
    DatabaseType.MYSQL,
    DatabaseType.MARIADB,
    DatabaseType.MONGODB,
  ])('updates the inline total when only metadata changes with %s', async (databaseType) => {
    let totalUsageMb = 104;
    const backupRows = [buildBackup('backup-id', 'COMPLETED')];
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get('*/api/v1/logical-backups', () =>
        HttpResponse.json<LogicalBackupPage>({ items: backupRows, totalUsageMb }),
      ),
    );
    vi.useFakeTimers();
    renderScreen(
      <LogicalBackupsComponent
        database={{ ...backedUpDatabase, settings: createDefaultDatabaseSettings(databaseType) }}
        isCanManageDBs={false}
      />,
    );
    await waitForCondition(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*104 MB/ })).toBeTruthy(),
    );

    totalUsageMb = 10752;
    await advanceClockUntil(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*10.5 GB/ })).toBeTruthy(),
    );

    totalUsageMb = 0;
    await advanceClockUntil(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*0 MB/ })).toBeTruthy(),
    );
  });

  it('hides the previous total and waits for explicit retry when reloading after deletion fails', async () => {
    let hasBackupBeenDeleted = false;
    let hasReadRecovered = false;
    let requestCount = 0;
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get('*/api/v1/logical-backups', () => {
        requestCount++;

        if (hasReadRecovered)
          return HttpResponse.json<LogicalBackupPage>({ items: [], totalUsageMb: 0 });

        if (hasBackupBeenDeleted)
          return HttpResponse.json(buildProblem('internal', 500), { status: 500 });

        return HttpResponse.json<LogicalBackupPage>({
          items: [{ ...buildBackup('deleted-backup', 'COMPLETED'), sizeMb: 104 }],
          totalUsageMb: 104,
        });
      }),
      http.delete('*/api/v1/logical-backups/:backupId', () => {
        hasBackupBeenDeleted = true;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    vi.useFakeTimers();
    renderScreen(<LogicalBackupsComponent database={backedUpDatabase} isCanManageDBs />);
    await waitForCondition(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*104 MB/ })).toBeTruthy(),
    );

    fireEvent.click(screen.getAllByRole('button', { name: 'Delete backup' })[0]);
    await waitForCondition(() =>
      expect(screen.getByRole('button', { name: 'Delete' })).toBeTruthy(),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    await waitForCondition(() =>
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy(),
    );
    expect(screen.queryByRole('columnheader', { name: /^Size.*MB/ })).toBeNull();
    expect(screen.queryByText('Total usage: 104 MB')).toBeNull();
    const requestsBeforeRetry = requestCount;

    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    fireEvent(document, new Event('visibilitychange'));
    await advanceFakeClock(3000);
    expect(requestCount).toBe(requestsBeforeRetry);

    visibility.mockReturnValue('visible');
    fireEvent(document, new Event('visibilitychange'));
    await advanceFakeClock(3000);
    expect(requestCount).toBe(requestsBeforeRetry);
    expect(screen.queryByRole('columnheader', { name: /^Size.*MB/ })).toBeNull();
    expect(screen.queryByText('Total usage: 104 MB')).toBeNull();

    hasReadRecovered = true;
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await waitForCondition(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*0 MB/ })).toBeTruthy(),
    );
    await advanceClockUntil(() => expect(requestCount).toBeGreaterThan(requestsBeforeRetry + 1));
    expect(screen.queryByRole('columnheader', { name: /^Size.*104 MB/ })).toBeNull();
  });

  it('requires explicit retry after a failed first page despite polling and visibility changes', async () => {
    let requestCount = 0;
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get('*/api/v1/logical-backups', () => {
        requestCount++;
        return HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 });
      }),
    );
    vi.useFakeTimers();
    renderScreen(<LogicalBackupsComponent database={backedUpDatabase} isCanManageDBs={false} />);
    await waitForCondition(() =>
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy(),
    );

    fireEvent(document, new Event('visibilitychange'));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000);
    });

    expect(requestCount).toBe(1);
    expect(screen.queryByRole('columnheader', { name: /^Size.*MB/ })).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await waitForCondition(() => expect(requestCount).toBe(2));
  });
});
