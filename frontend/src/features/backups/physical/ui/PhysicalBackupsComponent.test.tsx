// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { BackupConfig } from '../../../../entities/backup-configs';
import { DatabaseType, createDefaultDatabaseSettings } from '../../../../entities/databases';
import { DICTIONARIES } from '../../../../shared/i18n';
import { advanceClockUntil } from '../../../../shared/lib/test/advanceClockUntil';
import { waitForCondition } from '../../../../shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { PhysicalBackupsComponent } from './PhysicalBackupsComponent';

dayjs.extend(utc);

const server = setUpMockApi();
const database = {
  id: 'old-database',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  notifierIds: [],
};
const config: BackupConfig = {
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: {
    type: 'PHYSICAL',
    fullBackupSchedule: { type: 'HOURLY' },
    incrementalBackupSchedule: null,
    retention: {
      policy: 'FULL_BACKUPS',
      chainsCount: 0,
      fullBackups: {
        policy: 'LAST_N',
        count: 5,
        gfsHours: 0,
        gfsDays: 0,
        gfsWeeks: 0,
        gfsMonths: 0,
        gfsYears: 0,
      },
    },
    walLagThresholdBytes: 0,
    notificationTriggers: [],
  },
};
const usageText = (size: number) =>
  DICTIONARIES.en.backups.physical.list.totalUsage.replace('{{size}}', `${size} MB`);

describe('PhysicalBackupsComponent', () => {
  it('keeps the current usage when an old refresh answers after the replacement list loads', async () => {
    const oldRefresh = createPendingApiResponse();
    let oldRequestCount = 0;

    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      http.get('*/api/v1/physical-backups', ({ request }) => {
        if (new URL(request.url).searchParams.get('databaseId') === 'new-database') {
          return HttpResponse.json({ items: [], totalUsageMb: 30 });
        }
        oldRequestCount++;
        return oldRequestCount === 1
          ? HttpResponse.json({ items: [], totalUsageMb: 10 })
          : oldRefresh.response;
      }),
    );

    vi.useFakeTimers();
    const view = renderScreen(
      <PhysicalBackupsComponent database={database} isCanManageDBs={false} />,
    );

    await waitForCondition(() => expect(screen.getByText(usageText(10))).toBeTruthy());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000);
    });
    await waitForCondition(() => expect(oldRequestCount).toBe(2));

    view.rerender(
      <PhysicalBackupsComponent
        database={{ ...database, id: 'new-database' }}
        isCanManageDBs={false}
      />,
    );
    await waitForCondition(() => expect(screen.getByText(usageText(30))).toBeTruthy());

    const oldResponseProcessed = new Promise<void>((resolve) => {
      server.events.on('response:mocked', ({ request }) => {
        if (new URL(request.url).searchParams.get('databaseId') === 'old-database') resolve();
      });
    });

    await act(async () => {
      oldRefresh.respond(HttpResponse.json({ items: [], totalUsageMb: 20 }));
      await oldResponseProcessed;
    });

    expect(screen.getByText(usageText(30))).toBeTruthy();
    expect(screen.queryByText(usageText(20))).toBeNull();
  });

  describe('with a failing refresh', () => {
    const physicalBackup = {
      id: 'running-backup',
      databaseId: database.id,
      storageId: 'storage-id',
      kind: 'FULL',
      status: 'IN_PROGRESS',
      sizeMb: 12,
      durationMs: 3_000,
      encryption: 'NONE',
      createdAt: '2026-10-27T10:00:00Z',
    };

    const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });

    const countListRequests = () =>
      server.getRequests().filter((request) => request.url.includes('/physical-backups?')).length;

    it('shows no alert, keeps the rows and requests again when polls fail', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let listRequests = 0;
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () =>
          ++listRequests === 1
            ? HttpResponse.json({ items: [physicalBackup], totalUsageMb: 10 })
            : failedResponse(),
        ),
      );
      vi.useFakeTimers();
      renderScreen(<PhysicalBackupsComponent database={database} isCanManageDBs={false} />);
      await waitForCondition(() => screen.getByText(usageText(10)));
      const shownRunningBackups = screen.getAllByText('In progress').length;

      await advanceClockUntil(() => expect(countListRequests()).toBeGreaterThanOrEqual(3));

      expect(showAlert).not.toHaveBeenCalled();
      expect(screen.getByText(usageText(10))).toBeTruthy();
      expect(shownRunningBackups).toBeGreaterThan(0);
      expect(screen.getAllByText('In progress')).toHaveLength(shownRunningBackups);
    });

    it('shows no alert and refreshes again when the refresh after a successful cancel fails', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let isCancelled = false;
      let refreshesAfterCancel = 0;
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () => {
          if (!isCancelled) {
            return HttpResponse.json({ items: [physicalBackup], totalUsageMb: 10 });
          }

          refreshesAfterCancel += 1;
          return refreshesAfterCancel === 1
            ? failedResponse()
            : HttpResponse.json({
                items: [{ ...physicalBackup, status: 'CANCELED' }],
                totalUsageMb: 20,
              });
        }),
        http.post('*/api/v1/physical-backups/:backupId/cancel', () => {
          isCancelled = true;
          return HttpResponse.json({ ...physicalBackup, status: 'CANCELED' }, { status: 202 });
        }),
      );
      vi.useFakeTimers();
      renderScreen(<PhysicalBackupsComponent database={database} isCanManageDBs />);
      await waitForCondition(() => screen.getByText(usageText(10)));

      fireEvent.click(screen.getAllByRole('img', { name: 'close-circle' })[0]);

      await advanceClockUntil(() => screen.getByText(usageText(20)));

      expect(refreshesAfterCancel).toBeGreaterThanOrEqual(2);
      expect(showAlert).not.toHaveBeenCalled();
    });
  });
});
