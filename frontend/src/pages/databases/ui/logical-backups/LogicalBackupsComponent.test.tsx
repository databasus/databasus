import { fireEvent, screen, waitFor } from '@testing-library/react';
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
import { API_BASE_PATH } from '../../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
// @vitest-environment happy-dom
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
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
    const scrollContainer = renderBackupList();
    await screen.findAllByText('In progress');
    fireEvent.scroll(scrollContainer);
    await screen.findAllByText('Failed');

    expect(await screen.findAllByText('Successful')).toHaveLength(2);
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

    fireEvent.click(screen.getAllByRole('img', { name: 'download' })[0]);

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
      fireEvent.click(screen.getAllByRole('img', { name: 'safety' })[0]);
      fireEvent.click(await screen.findByRole('button', { name: 'Queue check' }));
      expect(await screen.findByText(text)).toBeTruthy();
    },
  );
});
