// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { describe, expect, it } from 'vitest';

import type { LogicalBackupConfigInput } from '../../../../entities/backups/config';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { EditLogicalBackupConfigComponent } from './EditLogicalBackupConfigComponent';

dayjs.extend(utc);

setUpMockApi();

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
    const offeredStorageIds: (string | null | undefined)[] = [];
    renderScreen(
      <EditLogicalBackupConfigComponent
        renderStorageSelection={({ selectedStorageId }) => {
          offeredStorageIds.push(selectedStorageId);
          return null;
        }}
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
    expect(offeredStorageIds[offeredStorageIds.length - 1]).toBe('archive-storage-id');
  });
});
