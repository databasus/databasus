// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  BackupConfig,
  BackupConfigInput,
  LogicalBackupConfigInput,
} from '@/entities/backup-configs';
import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { EditLogicalBackupConfigSection } from '@/pages/databases/backup-config/ui/EditLogicalBackupConfigSection';
import { API_BASE_PATH } from '@/shared/api';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

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
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: logicalSettings,
};

const chooseStorage = async (name = 'Archive') => {
  const storageLabel = await screen.findByText('Storage');
  const select = storageLabel.parentElement!.querySelector('[role="combobox"]')!;

  fireEvent.click(select);
  const option = await waitFor(() => {
    const storageOption = screen.queryByRole('option', { name: `${name}` });
    expect(storageOption).toBeTruthy();
    return storageOption!;
  });

  fireEvent.click(option);
};

const getFieldInput = (label: string) => screen.findByLabelText<HTMLInputElement>(label);

const getSaveButton = () => screen.findByRole('button', { name: /Save$/ });

const chooseOption = async ({
  fieldName,
  optionTitle,
}: {
  fieldName: string;
  optionTitle: string;
}) => {
  const control = await screen.findByRole('combobox', { name: fieldName });

  fireEvent.click(control!);

  const option = await waitFor(() => {
    const visibleOption = screen.queryByRole('option', { name: `${optionTitle}` });
    expect(visibleOption).toBeTruthy();
    return visibleOption!;
  });

  fireEvent.click(option);
};

const renderConfig = (
  schedule: LogicalBackupConfigInput['settings']['schedule'],
  retention = logicalSettings.retention,
  notificationTriggers = logicalSettings.notificationTriggers,
) => {
  const submitted: LogicalBackupConfigInput[] = [];

  mockServer.use(
    mockApi.get('/backup-configs/{databaseId}', 200, {
      ...config,
      settings: { ...logicalSettings, schedule, retention, notificationTriggers },
    }),
    mockApi.get('/logical-backups', 200, { totalUsageMb: 0, items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
    http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
      const saved = (await request.json()) as LogicalBackupConfigInput;
      submitted.push(saved);

      return HttpResponse.json({ ...saved, databaseId: database.id });
    }),
  );

  renderScreen(
    <EditLogicalBackupConfigSection database={database} onCancel={() => {}} onSaved={() => {}} />,
  );

  return submitted;
};

const saveAndGetSubmittedSettings = async (submitted: LogicalBackupConfigInput[]) => {
  fireEvent.click(await getSaveButton());
  await waitFor(() => expect(submitted).toHaveLength(1));

  return submitted[0].settings;
};

describe('EditLogicalBackupConfigSection', () => {
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
        totalUsageMb: 0,
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
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
        submitted.push((await request.json()) as BackupConfigInput);
        return HttpResponse.json({
          ...config,
          cleanup: { pendingFileCount: 2, failedFileCount: 0 },
        });
      }),
    );
    renderScreen(
      <EditLogicalBackupConfigSection database={database} onCancel={() => {}} onSaved={onSaved} />,
    );

    await chooseStorage();
    fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));

    expect(
      await screen.findByText(/Switching storage immediately removes the old backup history/),
    ).toBeTruthy();

    const warning = screen.getByText(
      /Switching storage immediately removes the old backup history/,
    );

    expect(warning.textContent).toContain('activates the new storage.');
    expect(warning.textContent).toContain('Unreachable storage delays cleanup.');
    expect(submitted).toHaveLength(0);

    fireEvent.click(screen.getByRole('button', { name: 'Switch storage and remove history' }));

    await waitFor(() => expect(submitted).toHaveLength(1));
    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1));
    expect(submitted).toHaveLength(1);
  });

  it('shows a refused save in the form with the entered values', async () => {
    const onSaved = vi.fn();
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      mockApi.get('/logical-backups', 200, { totalUsageMb: 0, items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
      mockApi.put('/backup-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );
    renderScreen(
      <EditLogicalBackupConfigSection database={database} onCancel={() => {}} onSaved={onSaved} />,
    );
    const saveButton = await screen.findByRole('button', { name: /Save$/ });
    fireEvent.click(screen.getAllByRole('checkbox')[0]);

    fireEvent.click(saveButton);

    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(true);
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', false);
    expect(onSaved).not.toHaveBeenCalled();
  });

  it('accepts a new retention count typed after the field is cleared, and shows the minimum after leaving it empty', async () => {
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      mockApi.get('/logical-backups', 200, { totalUsageMb: 0, items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
    );
    const { user } = renderScreen(
      <EditLogicalBackupConfigSection database={database} onCancel={() => {}} onSaved={() => {}} />,
    );
    const countField = await screen.findByLabelText<HTMLInputElement>('Most recent backups');

    await user.clear(countField);

    expect(countField.value).toBe('');

    await user.type(countField, '9');

    expect(countField.value).toBe('9');

    await user.clear(countField);
    await user.tab();

    expect(countField.value).toBe('1');
  });

  describe('when a saved configuration changes', () => {
    const ENCRYPTED_OPTION = 'Encrypt backup files';

    it('sends the chosen encryption when encryption changes', async () => {
      const submitted = renderConfig({ type: 'HOURLY' });

      await chooseOption({ fieldName: 'Encryption', optionTitle: ENCRYPTED_OPTION });
      await saveAndGetSubmittedSettings(submitted);

      expect(submitted[0].encryption).toBe('ENCRYPTED');
    });

    it('sends the checked triggers without the unchecked one when notification triggers change', async () => {
      const submitted = renderConfig({ type: 'HOURLY' }, logicalSettings.retention, [
        'BACKUP_FAILED',
      ]);

      fireEvent.click(await screen.findByRole('checkbox', { name: 'Backup success' }));
      fireEvent.click(screen.getByRole('checkbox', { name: 'Backup failed' }));
      const submittedSettings = await saveAndGetSubmittedSettings(submitted);

      expect(submittedSettings.notificationTriggers).toEqual(['BACKUP_SUCCESS']);
    });

    it('disables Save again when encryption returns to the saved value', async () => {
      renderConfig({ type: 'HOURLY' });

      expect(await getSaveButton()).toHaveProperty('disabled', true);

      await chooseOption({ fieldName: 'Encryption', optionTitle: ENCRYPTED_OPTION });

      expect(await getSaveButton()).toHaveProperty('disabled', false);

      await chooseOption({ fieldName: 'Encryption', optionTitle: 'None' });

      expect(await getSaveButton()).toHaveProperty('disabled', true);
    });

    it('disables Save again when notification triggers return to the saved set in another order', async () => {
      renderConfig({ type: 'HOURLY' }, logicalSettings.retention, [
        'BACKUP_SUCCESS',
        'BACKUP_FAILED',
      ]);
      const successTrigger = await screen.findByRole('checkbox', { name: 'Backup success' });

      fireEvent.click(successTrigger);

      expect(await getSaveButton()).toHaveProperty('disabled', false);

      fireEvent.click(successTrigger);

      expect(await getSaveButton()).toHaveProperty('disabled', true);
    });

    it('sends the entered daily, weekly, monthly and yearly values when the GFS policy is chosen', async () => {
      const submitted = renderConfig({ type: 'HOURLY' });

      await chooseOption({
        fieldName: 'Retention policy',
        optionTitle: 'GFS (calendar retention)',
      });
      fireEvent.change(await getFieldInput('Daily backups'), { target: { value: '11' } });
      fireEvent.change(await getFieldInput('Weekly backups'), { target: { value: '12' } });
      fireEvent.change(await getFieldInput('Monthly backups'), { target: { value: '13' } });
      fireEvent.change(await getFieldInput('Yearly backups'), { target: { value: '14' } });
      const submittedSettings = await saveAndGetSubmittedSettings(submitted);

      expect(submittedSettings.retention).toMatchObject({
        policy: 'GFS',
        gfsDays: 11,
        gfsWeeks: 12,
        gfsMonths: 13,
        gfsYears: 14,
      });
    });

    it('sends the chosen period when the time period policy is chosen', async () => {
      const submitted = renderConfig({ type: 'HOURLY' });

      await chooseOption({
        fieldName: 'Retention policy',
        optionTitle: 'Time period (last N days)',
      });
      await chooseOption({ fieldName: 'Period', optionTitle: '6 months' });
      const submittedSettings = await saveAndGetSubmittedSettings(submitted);

      expect(submittedSettings.retention).toMatchObject({
        policy: 'TIME_PERIOD',
        period: 'SIX_MONTHS',
      });
    });
  });

  describe('with a schedule', () => {
    const gfsRetention: LogicalBackupConfigInput['settings']['retention'] = {
      ...logicalSettings.retention,
      policy: 'GFS',
      gfsHours: 5,
      gfsDays: 7,
    };

    const chooseScheduleType = (optionTitle: string) =>
      chooseOption({ fieldName: 'Backup interval', optionTitle });

    const chooseTime = async (time: string) => {
      const timeInput = await getFieldInput('Time of day');

      fireEvent.click(timeInput);
      fireEvent.paste(timeInput, { clipboardData: { getData: () => time } });
      fireEvent.keyDown(timeInput, { key: 'Enter' });
    };

    it('names the retention period field', async () => {
      renderConfig(
        { type: 'HOURLY' },
        { ...logicalSettings.retention, policy: 'TIME_PERIOD', period: 'MONTH' },
      );

      expect(await screen.findByRole('combobox', { name: 'Period' })).toBeTruthy();
    });

    const renderMonthlyConfig = (dayOfMonth: number) =>
      renderConfig({ type: 'MONTHLY', timeOfDay: '12:00', dayOfMonth });

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

      fireEvent.click(timeInput);
      fireEvent.paste(timeInput, { clipboardData: { getData: () => '1:00 PM' } });
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
        weekdayControl.closest('div')!.querySelector('button[role="combobox"]')?.textContent,
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
