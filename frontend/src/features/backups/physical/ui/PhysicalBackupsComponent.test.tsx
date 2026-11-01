// @vitest-environment happy-dom
import { act, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { BackupConfig } from '../../../../entities/backups/config';
import { DatabaseType, createDefaultDatabaseSettings } from '../../../../entities/databases';
import { DICTIONARIES } from '../../../../shared/i18n';
import { waitForCondition } from '../../../../shared/lib/test/conditionWait';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { PhysicalBackupsComponent } from './PhysicalBackupsComponent';

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
});
