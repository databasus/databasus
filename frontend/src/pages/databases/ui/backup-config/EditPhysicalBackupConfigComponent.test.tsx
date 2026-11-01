import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  BackupConfig,
  BackupConfigInput,
  PhysicalBackupSettings,
  StorageSwitch,
} from '../../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import type { Schedule } from '../../../../entities/schedules';
import { API_BASE_PATH } from '../../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
// @vitest-environment happy-dom
import { EditPhysicalBackupConfigComponent } from './EditPhysicalBackupConfigComponent';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
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
const operation: StorageSwitch = {
  id: 'switch-id',
  databaseId: database.id,
  workspaceId: database.workspaceId,
  targetStorageId: 'target-storage',
  status: 'CLEANUP_PENDING',
  createdAt: '2026-10-30T10:00:00Z',
  cleanupDeadline: '2026-10-31T10:00:00Z',
  total: 2,
  pending: 2,
  deleted: 0,
  remaining: 0,
};

const chooseStorage = async (name = 'Archive') => {
  const storageLabel = await screen.findByText('Storage');
  const select = storageLabel.parentElement!.querySelector('.ant-select-selector')!;

  fireEvent.mouseDown(select);
  const option = await waitFor(() => {
    const storageOption = document.querySelector(`.ant-select-item-option[title="${name}"]`);
    expect(storageOption).toBeTruthy();
    return storageOption!;
  });

  fireEvent.click(option);
};

const incrementalDatabase: Database = {
  ...database,
  settings: {
    ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
    backupType: 'FULL_INCREMENTAL',
  } as Database['settings'],
};

const renderIncrementalConfig = ({
  fullBackupSchedule,
  incrementalBackupSchedule,
}: {
  fullBackupSchedule: Schedule;
  incrementalBackupSchedule: Schedule | null;
}) => {
  const submitted: BackupConfigInput[] = [];

  mockServer.use(
    mockApi.get('/backup-configs/{databaseId}', 200, {
      ...config,
      storageId: 'old-storage',
      settings: {
        ...config.settings,
        fullBackupSchedule,
        incrementalBackupSchedule,
        retention: { policy: 'CHAINS', chainsCount: 3, fullBackups: null },
      },
    } as BackupConfig),
    mockApi.get('/storages', 200, { items: [] }),
    mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
    mockApi.get('/storage-switches', 200, { items: [] }),
    http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
      const saved = (await request.json()) as BackupConfigInput;
      submitted.push(saved);

      return HttpResponse.json({ ...saved, databaseId: database.id });
    }),
  );

  renderScreen(
    <EditPhysicalBackupConfigComponent
      database={incrementalDatabase}
      isShowBackButton={false}
      onBack={() => {}}
      onCancel={() => {}}
      isSaveToApi
      onSaved={() => {}}
    />,
  );

  return submitted;
};

const findCadenceGroup = (name: string) => screen.findByRole('group', { name });

const getWeeklySchedule = (weekday: number): Schedule => ({
  type: 'WEEKLY',
  timeOfDay: '12:00',
  weekday,
});

describe('EditPhysicalBackupConfigComponent', () => {
  describe('with two schedules', () => {
    beforeEach(() => {
      vi.stubEnv('TZ', 'UTC');
    });

    afterEach(() => {
      vi.unstubAllEnvs();
    });

    it('labels the fields of each schedule separately', async () => {
      renderIncrementalConfig({
        fullBackupSchedule: getWeeklySchedule(1),
        incrementalBackupSchedule: getWeeklySchedule(2),
      });

      const fullGroup = await findCadenceGroup('Full backup cadence');
      const incrementalGroup = await findCadenceGroup('Incremental backup cadence');
      const fullWeekday = within(fullGroup).getByLabelText('Weekday');
      const incrementalWeekday = within(incrementalGroup).getByLabelText('Weekday');

      expect(fullWeekday).not.toBe(incrementalWeekday);
      expect(fullWeekday.id).not.toBe(incrementalWeekday.id);
    });

    it('offers only the type for an empty incremental schedule and keeps Save disabled', async () => {
      renderIncrementalConfig({
        fullBackupSchedule: { type: 'HOURLY' },
        incrementalBackupSchedule: null,
      });

      const incrementalGroup = await findCadenceGroup('Incremental backup cadence');

      expect(within(incrementalGroup).queryByLabelText('Time of day')).toBeNull();
      expect(within(incrementalGroup).queryByLabelText('Weekday')).toBeNull();

      fireEvent.click(screen.getAllByRole('checkbox')[0]);

      expect(screen.getByRole('button', { name: /Save$/ })).toHaveProperty('disabled', true);

      fireEvent.mouseDown(
        within(incrementalGroup)
          .getByRole('combobox', { name: 'Incremental backup cadence' })
          .closest('.ant-select')!
          .querySelector('.ant-select-selector')!,
      );

      const hourlyOption = await waitFor(() => {
        const visibleOption = document.querySelector('.ant-select-item-option[title="Hourly"]');
        expect(visibleOption).toBeTruthy();
        return visibleOption!;
      });

      fireEvent.click(hourlyOption);

      expect(screen.getByRole('button', { name: /Save$/ })).toHaveProperty('disabled', false);
    });

    it('lifts the schedule rule when backups are turned off', async () => {
      renderIncrementalConfig({
        fullBackupSchedule: { type: 'HOURLY' },
        incrementalBackupSchedule: null,
      });

      await findCadenceGroup('Incremental backup cadence');
      fireEvent.click(screen.getAllByRole('switch')[0]);

      expect(screen.getByRole('button', { name: /Save$/ })).toHaveProperty('disabled', false);
    });

    it('leaves the other schedule unchanged in the submitted config when one is edited', async () => {
      const submitted = renderIncrementalConfig({
        fullBackupSchedule: { type: 'MONTHLY', timeOfDay: '12:00', dayOfMonth: 5 },
        incrementalBackupSchedule: { type: 'MONTHLY', timeOfDay: '12:00', dayOfMonth: 6 },
      });
      const fullGroup = await findCadenceGroup('Full backup cadence');

      fireEvent.change(within(fullGroup).getByLabelText('Day of month'), {
        target: { value: '7' },
      });
      fireEvent.click(screen.getByRole('button', { name: /Save$/ }));
      await waitFor(() => expect(submitted).toHaveLength(1));

      const submittedSettings = submitted[0].settings as PhysicalBackupSettings;

      expect(submittedSettings.fullBackupSchedule.dayOfMonth).toBe(7);
      expect(submittedSettings.incrementalBackupSchedule).toEqual({
        type: 'MONTHLY',
        timeOfDay: '12:00',
        dayOfMonth: 6,
      });
    });
  });

  it('retains the storage selection and configuration draft when nested storage creation is canceled', async () => {
    const savedConfig = { ...config, storageId: 'old-storage', isBackupsEnabled: false };
    const onSaved = vi.fn();
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, { ...config, storageId: 'old-storage' }),
      mockApi.get('/storages', 200, {
        items: [
          {
            id: 'old-storage',
            workspaceId: database.workspaceId,
            name: 'Previous archive',
            settings: { type: 'LOCAL' },
          },
        ],
      }),
      mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.put('/backup-configs/{databaseId}', 200, savedConfig),
    );
    const view = renderScreen(
      <EditPhysicalBackupConfigComponent
        database={database}
        isShowBackButton={false}
        onBack={() => {}}
        onCancel={() => {}}
        isSaveToApi
        onSaved={onSaved}
      />,
    );
    await screen.findByText('Previous archive', { selector: '.ant-select-selection-item' });

    await view.user.click(screen.getByRole('switch'));
    await view.user.click(screen.getByRole('button', { name: 'Create new storage' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Add storage' }));
    await view.user.type(dialog.getByRole('textbox', { name: 'Name' }), 'Discarded nested archive');
    await view.user.click(dialog.getByRole('button', { name: 'Close' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(screen.getByRole('switch').getAttribute('aria-checked')).toBe('false');
    expect(
      screen.getByText('Previous archive', { selector: '.ant-select-selection-item' }),
    ).toBeTruthy();
    expect(onSaved).not.toHaveBeenCalled();
    expect(mockServer.getRequests().every((request) => request.method === 'GET')).toBe(true);

    await view.user.click(screen.getByRole('button', { name: /Save$/ }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1));
    const update = mockServer.getRequests().find((request) => request.method === 'PUT')!;

    expect(await update.clone().json()).toMatchObject({
      storageId: 'old-storage',
      isBackupsEnabled: false,
      settings: config.settings,
    });
    expect(onSaved).toHaveBeenCalledWith(
      expect.objectContaining({ storageId: 'old-storage', isBackupsEnabled: false }),
    );
  });

  it('requires confirmation before reassigning cleared storage with retained history and handles an accepted response', async () => {
    const submitted: BackupConfigInput[] = [];
    const onSaved = vi.fn();
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      mockApi.get('/storages', 200, {
        items: [
          {
            id: 'target-storage',
            workspaceId: database.workspaceId,
            name: 'Archive',
            settings: { type: 'LOCAL' },
          },
        ],
      }),
      mockApi.get('/physical-backups', 200, {
        items: [
          {
            id: 'old-backup',
            type: 'FULL',
            status: 'COMPLETED',
            timelineId: 1,
            startLsn: '0/1000000',
            stopLsn: '0/2000000',
            hasGapBefore: false,
            sizeMb: 1,
            createdAt: '2026-10-29T10:00:00Z',
          },
        ],
        totalUsageMb: 1,
      }),
      http.get(`*${API_BASE_PATH}/storage-switches`, () =>
        submitted.length
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json({ items: [] }),
      ),
      mockApi.get('/storage-switches/{storageSwitchId}', 200, operation),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
        submitted.push((await request.json()) as BackupConfigInput);
        return HttpResponse.json(operation, { status: 202 });
      }),
    );
    renderScreen(
      <EditPhysicalBackupConfigComponent
        database={database}
        isShowBackButton={false}
        onBack={() => {}}
        onCancel={() => {}}
        isSaveToApi
        onSaved={onSaved}
      />,
    );

    await chooseStorage();
    fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));

    expect(
      await screen.findByText(/Switching storage removes the old backup history immediately/),
    ).toBeTruthy();

    const warning = screen.getByText(
      /Switching storage removes the old backup history immediately/,
    );

    expect(warning.textContent).toContain('The new storage becomes active immediately.');
    expect(warning.textContent).toContain('If the old storage is unreachable, files may remain.');
    expect(submitted).toHaveLength(0);

    fireEvent.click(screen.getByRole('button', { name: 'Switch storage and remove history' }));

    await waitFor(() => expect(submitted).toHaveLength(1));
    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1));
    expect(
      await screen.findByText('Storage switched. Old files are being cleaned up.'),
    ).toBeTruthy();

    fireEvent.click(await screen.findByRole('button', { name: 'Retry status' }));

    expect(submitted).toHaveLength(1);
  });

  it.each(['old-storage', 'target-storage'])(
    'warns before reassignment to %s after clearing storage without remounting',
    async (storageId) => {
      const submitted: BackupConfigInput[] = [];
      const onSaved = vi.fn();
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, {
          ...config,
          storageId: 'old-storage',
        }),
        mockApi.get('/storages', 200, {
          items: [
            {
              id: 'old-storage',
              workspaceId: database.workspaceId,
              name: 'Previous archive',
              settings: { type: 'LOCAL' },
            },
            {
              id: 'target-storage',
              workspaceId: database.workspaceId,
              name: 'Archive',
              settings: { type: 'LOCAL' },
            },
          ],
        }),
        mockApi.get('/physical-backups', 200, {
          items: [
            {
              id: 'old-backup',
              type: 'FULL',
              status: 'COMPLETED',
              timelineId: 1,
              startLsn: '0/1000000',
              stopLsn: '0/2000000',
              hasGapBefore: false,
              sizeMb: 1,
              createdAt: '2026-10-29T10:00:00Z',
            },
          ],
          totalUsageMb: 1,
        }),
        mockApi.get('/storage-switches', 200, { items: [] }),
        mockApi.get('/storage-switches/{storageSwitchId}', 200, {
          ...operation,
          targetStorageId: storageId,
        }),
        http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
          const saved = (await request.json()) as BackupConfigInput;
          submitted.push(saved);

          return saved.storageId
            ? HttpResponse.json({ ...operation, targetStorageId: storageId }, { status: 202 })
            : HttpResponse.json({ ...saved, databaseId: database.id });
        }),
      );
      renderScreen(
        <EditPhysicalBackupConfigComponent
          database={database}
          isShowBackButton={false}
          onBack={() => {}}
          onCancel={() => {}}
          isSaveToApi
          onSaved={onSaved}
        />,
      );
      const storageLabel = await screen.findByText('Storage');
      const clear = storageLabel.parentElement!.querySelector('.ant-select-clear')!;

      expect(clear).toBeTruthy();

      fireEvent.mouseDown(clear);
      fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));

      await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1));
      expect(submitted).toHaveLength(1);
      expect(submitted[0].storageId).toBeNull();

      await chooseStorage(storageId === 'old-storage' ? 'Previous archive' : 'Archive');
      fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));

      expect(
        await screen.findByText(/Switching storage removes the old backup history immediately/),
      ).toBeTruthy();
      expect(submitted).toHaveLength(1);

      fireEvent.click(screen.getByRole('button', { name: 'Switch storage and remove history' }));

      await waitFor(() => expect(submitted).toHaveLength(2));
      expect(submitted[1].storageId).toBe(storageId);
    },
  );
});
