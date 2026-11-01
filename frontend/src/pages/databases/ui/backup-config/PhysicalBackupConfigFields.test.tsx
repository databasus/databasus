// @vitest-environment happy-dom
import { fireEvent, screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';

import type { PhysicalDatabaseBackupType } from '../../../../entities/databases';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { buildPhysicalBackupConfigDraft } from '../../model/physical-backups/buildPhysicalBackupConfigDraft.fixture';
import type { PhysicalBackupConfigDraft } from '../../model/physical-backups/PhysicalBackupConfigDraft';
import { PhysicalBackupConfigFields } from './PhysicalBackupConfigFields';

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
