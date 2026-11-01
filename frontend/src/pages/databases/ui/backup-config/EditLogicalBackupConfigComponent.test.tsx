import { fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  BackupConfig,
  BackupConfigInput,
  LogicalBackupConfigInput,
  StorageSwitch,
} from '../../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { API_BASE_PATH } from '../../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
// @vitest-environment happy-dom
import { EditLogicalBackupConfigComponent } from './EditLogicalBackupConfigComponent';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
};
const logicalSettings: LogicalBackupConfigInput['settings'] = {
  type: 'LOGICAL',
  schedule: { type: 'HOURLY' },
  retention: {
    policy: 'COUNT',
    count: 5,
    period: 'WEEK',
    gfsHours: 0,
    gfsDays: 0,
    gfsWeeks: 0,
    gfsMonths: 0,
    gfsYears: 0,
  },
  isRetryIfFailed: false,
  maxFailedTriesCount: 1,
  notificationTriggers: [],
};
const config: BackupConfig = {
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: logicalSettings,
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

const unsavedDatabase: Database = {
  id: '',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
  notifierIds: [],
};

const enteredConfig: LogicalBackupConfigInput = {
  isBackupsEnabled: true,
  storageId: 'archive-storage-id',
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'DAILY', timeOfDay: '04:00' },
    retention: {
      policy: 'COUNT',
      period: 'MONTH',
      count: 42,
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

describe('EditLogicalBackupConfigComponent', () => {
  it('starts an unsaved database from the given configuration instead of the defaults', async () => {
    mockServer.use(
      mockApi.get('/storages', 200, {
        items: [
          {
            id: 'archive-storage-id',
            workspaceId: database.workspaceId,
            name: 'Entered archive',
            settings: { type: 'LOCAL' },
          },
        ],
      }),
    );
    renderScreen(
      <EditLogicalBackupConfigComponent
        database={unsavedDatabase}
        initialConfig={enteredConfig}
        isShowBackButton={false}
        onBack={() => {}}
        onCancel={() => {}}
        isSaveToApi={false}
        onSaved={() => {}}
      />,
    );

    expect(await screen.findByDisplayValue('42')).toBeTruthy();
    expect(await screen.findByText('Entered archive')).toBeTruthy();
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
      mockApi.get('/logical-backups', 200, {
        items: [
          {
            id: 'old-backup',
            databaseId: database.id,
            storageId: 'old-storage',
            status: 'COMPLETED',
            sizeMb: 1,
            rawDatabaseSizeMb: 1,
            durationMs: 1,
            encryption: 'NONE',
            verificationStatus: 'NOT_VERIFIED',
            createdAt: '2026-10-29T10:00:00Z',
          },
        ],
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
      <EditLogicalBackupConfigComponent
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
        mockApi.get('/logical-backups', 200, {
          items: [
            {
              id: 'old-backup',
              databaseId: database.id,
              storageId: 'old-storage',
              status: 'COMPLETED',
              sizeMb: 1,
              rawDatabaseSizeMb: 1,
              durationMs: 1,
              encryption: 'NONE',
              verificationStatus: 'NOT_VERIFIED',
              createdAt: '2026-10-29T10:00:00Z',
            },
          ],
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
        <EditLogicalBackupConfigComponent
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

  describe('with a schedule', () => {
    const gfsRetention: LogicalBackupConfigInput['settings']['retention'] = {
      ...logicalSettings.retention,
      policy: 'GFS',
      gfsHours: 5,
      gfsDays: 7,
    };

    const getFieldInput = (label: string) => screen.findByLabelText<HTMLInputElement>(label);

    const getSaveButton = () => screen.findByRole('button', { name: /Save$/ });

    const chooseScheduleType = async (optionTitle: string) => {
      const control = await screen.findByRole('combobox', { name: 'Backup interval' });

      fireEvent.mouseDown(control.closest('.ant-select')!.querySelector('.ant-select-selector')!);

      const option = await waitFor(() => {
        const visibleOption = document.querySelector(
          `.ant-select-item-option[title="${optionTitle}"]`,
        );
        expect(visibleOption).toBeTruthy();
        return visibleOption!;
      });

      fireEvent.click(option);
    };

    const chooseTime = async (time: string) => {
      const timeInput = await getFieldInput('Time of day');

      fireEvent.mouseDown(timeInput);
      fireEvent.change(timeInput, { target: { value: time } });
      fireEvent.keyDown(timeInput, { key: 'Enter' });
    };

    const renderConfig = (
      schedule: LogicalBackupConfigInput['settings']['schedule'],
      retention = logicalSettings.retention,
    ) => {
      const submitted: LogicalBackupConfigInput[] = [];

      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, {
          ...config,
          settings: { ...logicalSettings, schedule, retention },
        }),
        mockApi.get('/logical-backups', 200, { items: [] }),
        mockApi.get('/storage-switches', 200, { items: [] }),
        mockApi.get('/storages', 200, { items: [] }),
        http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
          const saved = (await request.json()) as LogicalBackupConfigInput;
          submitted.push(saved);

          return HttpResponse.json({ ...saved, databaseId: database.id });
        }),
      );

      renderScreen(
        <EditLogicalBackupConfigComponent
          database={database}
          isShowBackButton={false}
          onBack={() => {}}
          onCancel={() => {}}
          isSaveToApi
          onSaved={() => {}}
        />,
      );

      return submitted;
    };

    const renderMonthlyConfig = (dayOfMonth: number) =>
      renderConfig({ type: 'MONTHLY', timeOfDay: '12:00', dayOfMonth });

    const saveAndGetSubmittedSettings = async (submitted: LogicalBackupConfigInput[]) => {
      fireEvent.click(await getSaveButton());
      await waitFor(() => expect(submitted).toHaveLength(1));

      return submitted[0].settings;
    };

    const saveAndGetSubmittedSchedule = async (submitted: LogicalBackupConfigInput[]) =>
      (await saveAndGetSubmittedSettings(submitted)).schedule;

    beforeEach(() => {
      vi.stubEnv('TZ', 'UTC');
      vi.useFakeTimers({ toFake: ['Date'] });
      vi.setSystemTime(new Date('2026-11-15T12:00:00Z'));
    });

    afterEach(() => {
      vi.unstubAllEnvs();
    });

    it('shows day 31 when a loaded config is on day 31', async () => {
      renderMonthlyConfig(31);

      const dayInput = await getFieldInput('Day of month');

      expect(dayInput.value).toBe('31');
    });

    it('submits day 31 when the user chooses day 31', async () => {
      const submitted = renderMonthlyConfig(10);
      const dayInput = await getFieldInput('Day of month');

      fireEvent.change(dayInput, { target: { value: '31' } });
      const submittedSchedule = await saveAndGetSubmittedSchedule(submitted);

      expect(submittedSchedule.dayOfMonth).toBe(31);
    });

    it('submits day 31 when the user changes only the time', async () => {
      const submitted = renderMonthlyConfig(31);
      const timeInput = await getFieldInput('Time of day');

      fireEvent.mouseDown(timeInput);
      fireEvent.change(timeInput, { target: { value: '1:00 PM' } });
      fireEvent.keyDown(timeInput, { key: 'Enter' });
      const submittedSchedule = await saveAndGetSubmittedSchedule(submitted);

      expect(submittedSchedule.timeOfDay).toBe('13:00');
      expect(submittedSchedule.dayOfMonth).toBe(31);
    });

    const fullyFilledHourlySchedule: LogicalBackupConfigInput['settings']['schedule'] = {
      type: 'HOURLY',
      timeOfDay: '12:00',
      weekday: 1,
      dayOfMonth: 1,
      cronExpression: '0 2 * * *',
    };

    it.each([
      ['Daily', 'DAILY'],
      ['Weekly', 'WEEKLY'],
      ['Monthly', 'MONTHLY'],
    ])(
      'sets the GFS hours to 24 when the type changes to %s',
      async (optionTitle, scheduleType) => {
        const submitted = renderConfig(fullyFilledHourlySchedule, gfsRetention);

        await chooseScheduleType(optionTitle);
        const submittedSettings = await saveAndGetSubmittedSettings(submitted);

        expect(submittedSettings.schedule.type).toBe(scheduleType);
        expect(submittedSettings.retention.gfsHours).toBe(24);
      },
    );

    it('keeps the GFS hours when the type changes to cron', async () => {
      const submitted = renderConfig(fullyFilledHourlySchedule, gfsRetention);

      await chooseScheduleType('Cron');
      const submittedSettings = await saveAndGetSubmittedSettings(submitted);

      expect(submittedSettings.schedule.type).toBe('CRON');
      expect(submittedSettings.retention.gfsHours).toBe(5);
    });

    it('keeps the GFS hours when only the time changes', async () => {
      const submitted = renderConfig({ type: 'DAILY', timeOfDay: '12:00' }, gfsRetention);

      await chooseTime('1:00 PM');
      const submittedSettings = await saveAndGetSubmittedSettings(submitted);

      expect(submittedSettings.schedule.timeOfDay).toBe('13:00');
      expect(submittedSettings.retention.gfsHours).toBe(5);
    });

    it('keeps Save disabled until an incomplete schedule gets its time', async () => {
      renderConfig({ type: 'HOURLY' });

      await chooseScheduleType('Daily');

      expect(await getSaveButton()).toHaveProperty('disabled', true);

      await chooseTime('4:00 AM');

      expect(await getSaveButton()).toHaveProperty('disabled', false);
    });

    it('names the fields of a weekly schedule without the word Backup', async () => {
      renderConfig({ type: 'WEEKLY', timeOfDay: '12:00', weekday: 1 });

      expect(await screen.findByLabelText('Weekday')).toBeTruthy();
      expect(screen.getByLabelText('Time of day')).toBeTruthy();
      expect(screen.queryByText('Backup weekday')).toBeNull();
      expect(screen.queryByText('Backup time of day')).toBeNull();
    });

    it('shows a saved weekday 0 as Sunday and allows saving', async () => {
      const submitted = renderConfig({ type: 'WEEKLY', timeOfDay: '12:00', weekday: 0 });
      const weekdayControl = await screen.findByRole('combobox', { name: 'Weekday' });

      expect(
        weekdayControl.closest('.ant-select')!.querySelector('.ant-select-selection-item')
          ?.textContent,
      ).toBe('Sun');

      fireEvent.click(screen.getAllByRole('checkbox')[0]);

      expect(await getSaveButton()).toHaveProperty('disabled', false);

      const submittedSchedule = await saveAndGetSubmittedSchedule(submitted);

      expect(submittedSchedule.weekday).toBe(0);
    });

    it('keeps Save disabled for a saved time zone prefix until the expression is replaced', async () => {
      const submitted = renderConfig({
        type: 'CRON',
        cronExpression: 'TZ=Europe/Moscow 0 2 * * *',
      });
      const cronInput = await getFieldInput('Cron expression (UTC)');

      expect(screen.getByText('Invalid cron expression')).toBeTruthy();

      fireEvent.click(screen.getAllByRole('checkbox')[0]);

      expect(await getSaveButton()).toHaveProperty('disabled', true);

      fireEvent.change(cronInput, { target: { value: '0 2 * * *' } });

      expect(await getSaveButton()).toHaveProperty('disabled', false);

      const submittedSchedule = await saveAndGetSubmittedSchedule(submitted);

      expect(submittedSchedule.cronExpression).toBe('0 2 * * *');
    });

    it('allows saving a saved time zone prefix after backups are turned off', async () => {
      const submitted = renderConfig({
        type: 'CRON',
        cronExpression: 'TZ=Europe/Moscow 0 2 * * *',
      });
      await getFieldInput('Cron expression (UTC)');

      fireEvent.click(screen.getAllByRole('switch')[0]);

      expect(await getSaveButton()).toHaveProperty('disabled', false);

      const submittedSettings = await saveAndGetSubmittedSettings(submitted);

      expect(submittedSettings.schedule.cronExpression).toBe('TZ=Europe/Moscow 0 2 * * *');
      expect(submitted[0].isBackupsEnabled).toBe(false);
    });
  });
});
