// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  BackupConfig,
  BackupConfigInput,
  PhysicalBackupSettings,
} from '../../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import type { Schedule } from '../../../../entities/schedules';
import { API_BASE_PATH } from '../../../../shared/api';
import { DICTIONARIES, useLocale } from '../../../../shared/i18n';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { EditPhysicalBackupConfigSection } from './EditPhysicalBackupConfigSection';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  notifierIds: [],
};
const config: BackupConfig = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
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
const incrementalDatabase: Database = {
  ...database,
  settings: {
    ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
    backupType: 'FULL_INCREMENTAL',
  } as Database['settings'],
};

const serveIncrementalSection = ({
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
    http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
      const saved = (await request.json()) as BackupConfigInput;
      submitted.push(saved);

      return HttpResponse.json({ ...saved, databaseId: database.id });
    }),
  );

  return submitted;
};

const IncrementalSection = () => (
  <EditPhysicalBackupConfigSection
    database={incrementalDatabase}
    onCancel={() => {}}
    onSaved={() => {}}
  />
);

const renderIncrementalSection = (schedules: {
  fullBackupSchedule: Schedule;
  incrementalBackupSchedule: Schedule | null;
}) => {
  const submitted = serveIncrementalSection(schedules);

  renderScreen(<IncrementalSection />);

  return submitted;
};

const IncrementalSectionWithLanguageSwitch = () => {
  const { setLocale } = useLocale();

  return (
    <>
      <button onClick={() => setLocale('fr')}>Change language</button>
      <IncrementalSection />
    </>
  );
};

const findCadenceGroup = (name: string) => screen.findByRole('group', { name });

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

const chooseEncryption = async (optionTitle: string) => {
  const control = await screen.findByRole('combobox', { name: 'Encryption' });

  fireEvent.mouseDown(control.closest('.ant-select')!.querySelector('.ant-select-selector')!);

  const option = await waitFor(() => {
    const visibleOption = document.querySelector(`.ant-select-item-option[title="${optionTitle}"]`);
    expect(visibleOption).toBeTruthy();
    return visibleOption!;
  });

  fireEvent.click(option);
};

const getSaveButton = () => screen.findByRole('button', { name: /Save$/ });

const saveAndGetSubmittedConfig = async (submitted: BackupConfigInput[]) => {
  fireEvent.click(await getSaveButton());
  await waitFor(() => expect(submitted).toHaveLength(1));

  return submitted[0];
};

describe('EditPhysicalBackupConfigSection', () => {
  describe('with two schedules', () => {
    beforeEach(() => {
      vi.stubEnv('TZ', 'UTC');
    });

    afterEach(() => {
      vi.unstubAllEnvs();
    });

    it('offers only the type for an empty incremental schedule and keeps Save disabled', async () => {
      renderIncrementalSection({
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
      renderIncrementalSection({
        fullBackupSchedule: { type: 'HOURLY' },
        incrementalBackupSchedule: null,
      });

      await findCadenceGroup('Incremental backup cadence');
      fireEvent.click(screen.getByRole('switch'));

      expect(screen.getByRole('button', { name: /Save$/ })).toHaveProperty('disabled', false);
    });

    it('leaves the other schedule unchanged in the submitted config when one is edited', async () => {
      const submitted = renderIncrementalSection({
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

  describe('when a saved configuration changes', () => {
    const ENCRYPTED_OPTION = 'Encrypt backup files';
    const hourlySchedules = {
      fullBackupSchedule: { type: 'HOURLY' },
      incrementalBackupSchedule: { type: 'HOURLY' },
    } as const;

    it('keeps the draft when the language changes', async () => {
      const submitted = serveIncrementalSection(hourlySchedules);
      renderScreen(<IncrementalSectionWithLanguageSwitch />);

      await chooseEncryption(ENCRYPTED_OPTION);
      fireEvent.click(screen.getByRole('button', { name: 'Change language' }));

      const translatedEncryption = await screen.findByRole('combobox', {
        name: DICTIONARIES.fr.backups.config.encryption,
      });
      const saveButton = screen.getByRole('button', {
        name: new RegExp(`${DICTIONARIES.fr.common.actions.save}$`),
      }) as HTMLButtonElement;

      expect(translatedEncryption.closest('.ant-select')?.textContent).toContain(
        DICTIONARIES.fr.backups.encryptionOptions.encrypted,
      );
      expect(saveButton.disabled).toBe(false);
      expect(submitted).toHaveLength(0);
    });

    it('sends the chosen encryption when encryption changes', async () => {
      const submitted = renderIncrementalSection(hourlySchedules);

      await chooseEncryption(ENCRYPTED_OPTION);
      const submittedConfig = await saveAndGetSubmittedConfig(submitted);

      expect(submittedConfig.encryption).toBe('ENCRYPTED');
    });

    it('sends the checked triggers of the backup type when notification triggers change', async () => {
      const submitted = renderIncrementalSection(hourlySchedules);

      fireEvent.click(await screen.findByRole('checkbox', { name: 'Backup failed' }));
      fireEvent.click(screen.getByRole('checkbox', { name: 'Chain broken' }));
      const submittedConfig = await saveAndGetSubmittedConfig(submitted);

      expect((submittedConfig.settings as PhysicalBackupSettings).notificationTriggers).toEqual([
        'BACKUP_FAILED',
        'CHAIN_BROKEN',
      ]);
    });

    it('disables Save again when encryption returns to the saved value', async () => {
      renderIncrementalSection(hourlySchedules);

      expect(await getSaveButton()).toHaveProperty('disabled', true);

      await chooseEncryption(ENCRYPTED_OPTION);

      expect(await getSaveButton()).toHaveProperty('disabled', false);

      await chooseEncryption('None');

      expect(await getSaveButton()).toHaveProperty('disabled', true);
    });

    it('disables Save again when notification triggers return to the saved set', async () => {
      renderIncrementalSection(hourlySchedules);
      const chainTrigger = await screen.findByRole('checkbox', { name: 'Chain broken' });

      fireEvent.click(chainTrigger);

      expect(await getSaveButton()).toHaveProperty('disabled', false);

      fireEvent.click(chainTrigger);

      expect(await getSaveButton()).toHaveProperty('disabled', true);
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
      mockApi.put('/backup-configs/{databaseId}', 200, savedConfig),
    );
    const view = renderScreen(
      <EditPhysicalBackupConfigSection database={database} onCancel={() => {}} onSaved={onSaved} />,
    );
    await screen.findByText('Previous archive', { selector: '.ant-select-selection-item' });

    await view.user.click(screen.getByRole('switch'));
    await view.user.click(await screen.findByRole('combobox', { name: 'Storage' }));
    await view.user.click(screen.getByRole('button', { name: 'Create new storage' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Add storage' }));
    await view.user.type(dialog.getByRole('textbox', { name: 'Name' }), 'Discarded nested archive');
    await view.user.click(dialog.getByRole('button', { name: 'Close' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByRole('combobox', { name: 'Storage' })),
    );
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
  });

  it('requires confirmation before reassigning cleared storage with retained history', async () => {
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
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
        submitted.push((await request.json()) as BackupConfigInput);
        return HttpResponse.json({
          ...config,
          cleanup: { pendingFileCount: 2, failedFileCount: 0 },
        });
      }),
    );
    renderScreen(
      <EditPhysicalBackupConfigSection database={database} onCancel={() => {}} onSaved={onSaved} />,
    );

    await chooseStorage();
    fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));

    const warning = await screen.findByText(
      /Switching storage immediately removes the old backup history/,
    );

    expect(warning.textContent).toContain('activates the new storage.');
    expect(warning.textContent).toContain('Unreachable storage delays cleanup.');
    expect(submitted).toHaveLength(0);

    fireEvent.click(screen.getByRole('button', { name: 'Switch storage and remove history' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1));
    expect(submitted).toHaveLength(1);
  });
});
