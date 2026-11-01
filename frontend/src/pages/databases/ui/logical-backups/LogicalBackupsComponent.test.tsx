import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useRef } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { BackupConfig } from '../../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { restoreApi } from '../../../../entities/restores';
import { verificationRunsApi } from '../../../../entities/verification/runs';
import { API_BASE_PATH } from '../../../../shared/api';
import { advanceClockUntil } from '../../../../shared/lib/test/advanceClockUntil';
import { waitForCondition } from '../../../../shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
// @vitest-environment happy-dom
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '../../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { logicalBackupApi } from '../../api/logicalBackupApi';
import type { LogicalBackup } from '../../model/LogicalBackup';
import type { LogicalBackupPage } from '../../model/LogicalBackupPage';
import { LogicalBackupsComponent } from './LogicalBackupsComponent';

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
function ScrollingDatabasePage() {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  return (
    <div ref={scrollContainerRef} className="database-scroll-container">
      <LogicalBackupsComponent
        database={backedUpDatabase}
        isCanManageDBs={true}
        scrollContainerRef={scrollContainerRef}
      />
    </div>
  );
}

const renderBackupList = () => {
  const { container } = renderScreen(<ScrollingDatabasePage />);

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

        return HttpResponse.json(
          cursor
            ? { items: [buildBackup('oldest-backup', 'FAILED')] }
            : {
                items: [buildBackup('newest-backup', 'COMPLETED')],
                nextCursor: 'first-page-cursor',
              },
        );
      }),
    );
    const scrollContainer = renderBackupList();
    await screen.findAllByText('Successful');

    fireEvent.scroll(scrollContainer);

    expect(await screen.findAllByText('Failed')).toHaveLength(2);
    expect(await screen.findAllByText('All backups loaded (2 total)')).toHaveLength(2);
    // The list also polls its first page, so polls may fall between these two requests.
    expect(requestedCursors[0]).toBeNull();
    expect(requestedCursors.filter((cursor) => cursor !== null)).toEqual(['first-page-cursor']);
  });

  it('refreshes the first page and keeps the scrolled tail when the poll answers', async () => {
    const requestedCursors: (string | null)[] = [];
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      http.get(`*${API_BASE_PATH}/logical-backups`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        if (cursor) return HttpResponse.json({ items: [buildBackup('oldest-backup', 'FAILED')] });

        return HttpResponse.json({
          items: [
            buildBackup(
              'newest-backup',
              requestedCursors.includes('first-page-cursor') ? 'COMPLETED' : 'IN_PROGRESS',
            ),
          ],
          nextCursor: 'first-page-cursor',
        });
      }),
    );
    vi.useFakeTimers();
    const scrollContainer = renderBackupList();
    await waitForCondition(() => screen.getAllByText('In progress'));
    fireEvent.scroll(scrollContainer);
    await waitForCondition(() => screen.getAllByText('Failed'));

    await advanceClockUntil(() => expect(screen.getAllByText('Successful')).toHaveLength(2));

    expect(screen.queryAllByText('In progress')).toHaveLength(0);
    expect(screen.getAllByText('Failed')).toHaveLength(2);
    expect(requestedCursors.filter((cursor) => cursor !== null)).toEqual(['first-page-cursor']);
    expect(requestedCursors[requestedCursors.length - 1]).toBeNull();
  });

  it('shows the started backup at once when a manual backup is started', async () => {
    const startedDatabaseIds: unknown[] = [];
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, backupConfigOfDatabase),
      mockBackupPages([{ items: [] }], []),
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
      mockBackupPages([{ items: [buildBackup('completed-backup', 'COMPLETED')] }], []),
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
        mockBackupPages([{ items: [buildBackup('completed-backup', 'COMPLETED')] }], []),
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
          ? HttpResponse.json({ items: [buildBackup('completed-backup', 'COMPLETED')] })
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
          : HttpResponse.json({ items: [buildBackup('completed-backup', 'COMPLETED')] }),
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
      mockBackupPages([{ items: [] }], []),
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
          HttpResponse.json({
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

      expect(
        screen.getByText('Restore from backup', { selector: '.ant-modal-title' }),
      ).toBeTruthy();
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
      mockBackupPages([{ items: [buildBackup('completed-backup', 'COMPLETED')] }], []),
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
        mockBackupPages([{ items: [buildBackup('listed-backup', status)] }], []),
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
            return HttpResponse.json({ items: [buildBackup('listed-backup', 'COMPLETED')] });
          }

          // A held poll keeps later polls off, so only the reload can change the list.
          return listRequests === 2
            ? createPendingApiResponse().response
            : HttpResponse.json({ items: [] });
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
            ? HttpResponse.json({ items: [buildBackup('listed-backup', 'IN_PROGRESS')] })
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
      // AntD ignores a click until it has ended the loading state of its button.
      await waitFor(() => expect(getDownloadButton().className).not.toContain('ant-btn-loading'));
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
          return HttpResponse.json({
            items: [buildBackup('newest-backup', 'COMPLETED')],
            nextCursor: 'first-page-cursor',
          });
        }

        return isNextPageFailing
          ? failedResponse()
          : HttpResponse.json({ items: [buildBackup('oldest-backup', 'FAILED')] });
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
