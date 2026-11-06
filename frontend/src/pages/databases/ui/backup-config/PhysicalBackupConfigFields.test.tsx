// @vitest-environment happy-dom
import { fireEvent, screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';

import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import { buildPhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/buildPhysicalBackupConfigDraft.fixture';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/PhysicalBackupConfigDraft';
import { PhysicalBackupConfigFields } from '@/pages/databases/ui/backup-config/PhysicalBackupConfigFields';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

interface FieldsParentProps {
  initialDraft: PhysicalBackupConfigDraft;
  backupType: PhysicalDatabaseBackupType;
}

interface OptionChoice {
  controlName: string;
  optionName: string;
}

const FieldsParent = ({ initialDraft, backupType }: FieldsParentProps) => {
  const [draft, setDraft] = useState(initialDraft);

  return (
    <PhysicalBackupConfigFields
      draft={draft}
      onChange={setDraft}
      workspaceId="workspace-id"
      backupType={backupType}
      canClearStorage
    />
  );
};

const renderFields = (
  backupType: PhysicalDatabaseBackupType = 'FULL_INCREMENTAL',
  initialDraft = buildPhysicalBackupConfigDraft({ backupType }),
) => {
  mockServer.use(mockApi.get('/storages', 200, { items: [] }));

  return renderScreen(<FieldsParent initialDraft={initialDraft} backupType={backupType} />);
};

const chooseOption = async (
  { user }: ReturnType<typeof renderFields>,
  { controlName, optionName }: OptionChoice,
) => {
  await user.click(screen.getByRole('combobox', { name: controlName }));
  await user.click(await screen.findByRole('option', { name: optionName }));
};

describe('PhysicalBackupConfigFields', () => {
  it.each(['FULL', 'FULL_INCREMENTAL', 'FULL_INCREMENTAL_WAL_STREAM'] as const)(
    'explains GFS without changing retention values for %s backups',
    async (backupType) => {
      const view = renderFields(backupType);
      const dailyCount = screen.getByRole('spinbutton', { name: 'Daily' });

      await view.user.clear(dailyCount);
      await view.user.type(dailyCount, '9');

      const hintButton = screen.getByRole('button', {
        name: 'What is GFS (Grandfather-Father-Son)?',
      });

      expect(hintButton.getAttribute('aria-expanded')).toBe('false');

      await view.user.click(hintButton);

      expect(hintButton.getAttribute('aria-expanded')).toBe('true');
      expect(screen.getByText(/GFS \(Grandfather-Father-Son\) rotation:/)).toBeTruthy();
      expect(dailyCount).toHaveProperty('value', '9');

      await view.user.click(screen.getByRole('button', { name: 'Hide' }));

      expect(screen.queryByText(/GFS \(Grandfather-Father-Son\) rotation:/)).toBeNull();
      expect(dailyCount).toHaveProperty('value', '9');

      await chooseOption(view, { controlName: 'Full backups', optionName: 'Count' });

      expect(
        screen.queryByRole('button', { name: 'What is GFS (Grandfather-Father-Son)?' }),
      ).toBeNull();
    },
  );

  it('labels the fields of each schedule separately', async () => {
    renderFields(
      'FULL_INCREMENTAL',
      buildPhysicalBackupConfigDraft({
        settings: {
          fullBackupSchedule: { type: 'WEEKLY', timeOfDay: '12:00', weekday: 1 },
          incrementalBackupSchedule: { type: 'WEEKLY', timeOfDay: '12:00', weekday: 2 },
        },
      }),
    );

    const fullGroup = await screen.findByRole('group', { name: 'Full backup cadence' });
    const incrementalGroup = screen.getByRole('group', { name: 'Incremental backup cadence' });
    const fullWeekday = within(fullGroup).getByLabelText('Weekday');
    const incrementalWeekday = within(incrementalGroup).getByLabelText('Weekday');

    expect(fullWeekday).not.toBe(incrementalWeekday);
    expect(fullWeekday.id).not.toBe(incrementalWeekday.id);
  });

  it('names every retention control', () => {
    renderFields();

    expect(screen.getByRole('combobox', { name: 'Retention' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Full backups' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Encryption' })).toBeTruthy();
    expect(screen.getByRole('spinbutton', { name: 'Chains' })).toBeTruthy();

    for (const period of ['Hourly', 'Daily', 'Weekly', 'Monthly', 'Yearly']) {
      expect(screen.getByRole('spinbutton', { name: period })).toBeTruthy();
    }
  });

  it.each([
    { fieldName: 'Chains', enteredValue: '9', minimum: '1' },
    { fieldName: 'Daily', enteredValue: '9', minimum: '0' },
  ])(
    'accepts a new $fieldName value typed after the field is cleared, and shows the minimum after leaving it empty',
    async ({ fieldName, enteredValue, minimum }) => {
      const { user } = renderFields();
      const field = screen.getByRole('spinbutton', { name: fieldName });

      await user.clear(field);

      expect(field).toHaveProperty('value', '');

      await user.type(field, enteredValue);

      expect(field).toHaveProperty('value', enteredValue);

      await user.clear(field);
      await user.tab();

      expect(field).toHaveProperty('value', minimum);
    },
  );

  it('shows the entered values after a switch of the retention policy and back', async () => {
    const view = renderFields();

    fireEvent.change(screen.getByRole('spinbutton', { name: 'Chains' }), {
      target: { value: '11' },
    });
    fireEvent.change(screen.getByRole('spinbutton', { name: 'Daily' }), {
      target: { value: '22' },
    });

    await chooseOption(view, { controlName: 'Retention', optionName: 'Keep last N chains' });

    expect(screen.queryByRole('spinbutton', { name: 'Daily' })).toBeNull();

    await chooseOption(view, {
      controlName: 'Retention',
      optionName: 'Keep last N chains and full backups',
    });

    expect(screen.getByRole('spinbutton', { name: 'Chains' })).toHaveProperty('value', '11');
    expect(screen.getByRole('spinbutton', { name: 'Daily' })).toHaveProperty('value', '22');
  });

  it('offers no retention policy and no chain count for a full-only backup type', () => {
    renderFields('FULL');

    expect(screen.queryByRole('combobox', { name: 'Retention' })).toBeNull();
    expect(screen.queryByRole('spinbutton', { name: 'Chains' })).toBeNull();
    expect(screen.getByRole('combobox', { name: 'Full backups' })).toBeTruthy();
    expect(screen.queryByRole('group', { name: 'Incremental backup cadence' })).toBeNull();
  });

  it.each([
    { backupType: 'FULL', triggerCount: 2 },
    { backupType: 'FULL_INCREMENTAL', triggerCount: 3 },
    { backupType: 'FULL_INCREMENTAL_WAL_STREAM', triggerCount: 4 },
  ] as const)('offers $triggerCount triggers for $backupType', ({ backupType, triggerCount }) => {
    renderFields(backupType);

    expect(screen.getAllByRole('checkbox')).toHaveLength(triggerCount);
  });

  it('hides the schedules and the triggers and keeps the retention with backups off', () => {
    renderFields(
      'FULL_INCREMENTAL',
      buildPhysicalBackupConfigDraft({ config: { isBackupsEnabled: false } }),
    );

    expect(screen.queryByRole('group', { name: 'Full backup cadence' })).toBeNull();
    expect(screen.queryByRole('checkbox')).toBeNull();
    expect(screen.getByRole('spinbutton', { name: 'Chains' })).toBeTruthy();
  });
});
