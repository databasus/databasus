// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { LogicalBackupConfigInput } from '@/entities/backup-configs';
import { CreateLogicalBackupConfigStep } from '@/pages/databases/ui/backup-config/CreateLogicalBackupConfigStep';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

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

const declareStorages = () =>
  mockServer.use(
    mockApi.get('/storages', 200, {
      items: [
        {
          id: 'archive-storage-id',
          workspaceId: 'workspace-id',
          name: 'Entered archive',
          settings: { type: 'LOCAL' },
        },
      ],
    }),
  );

describe('CreateLogicalBackupConfigStep', () => {
  it('starts from the given configuration instead of the defaults', async () => {
    declareStorages();
    renderScreen(
      <CreateLogicalBackupConfigStep
        workspaceId="workspace-id"
        initialConfig={enteredConfig}
        onBack={() => {}}
        onContinue={() => {}}
      />,
    );

    expect(await screen.findByDisplayValue('42')).toBeTruthy();
    expect(await screen.findByText('Entered archive')).toBeTruthy();
  });

  it('keeps Continue disabled without a storage', async () => {
    mockServer.use(mockApi.get('/storages', 200, { items: [] }));
    renderScreen(
      <CreateLogicalBackupConfigStep
        workspaceId="workspace-id"
        onBack={() => {}}
        onContinue={() => {}}
      />,
    );
    await screen.findByText('Storage');

    expect(screen.getByRole('button', { name: 'Continue' })).toHaveProperty('disabled', true);
  });

  it('opens the advanced settings for a draft with values other than the defaults', async () => {
    declareStorages();
    renderScreen(
      <CreateLogicalBackupConfigStep
        workspaceId="workspace-id"
        initialConfig={enteredConfig}
        onBack={() => {}}
        onContinue={() => {}}
      />,
    );

    expect(await screen.findByText('Retry backup if failed')).toBeTruthy();
  });

  it('keeps the advanced settings closed for a new draft', async () => {
    mockServer.use(mockApi.get('/storages', 200, { items: [] }));
    renderScreen(
      <CreateLogicalBackupConfigStep
        workspaceId="workspace-id"
        onBack={() => {}}
        onContinue={() => {}}
      />,
    );
    await screen.findByText('Storage');

    expect(screen.queryByText('Retry backup if failed')).toBeNull();
  });

  it('offers no Back without a way to go back', async () => {
    declareStorages();
    renderScreen(
      <CreateLogicalBackupConfigStep
        workspaceId="workspace-id"
        initialConfig={enteredConfig}
        onContinue={() => {}}
      />,
    );
    await screen.findByText('Entered archive');

    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull();
  });

  it('hands over the draft on Back and on Continue', async () => {
    declareStorages();
    const onBack = vi.fn();
    const onContinue = vi.fn();
    renderScreen(
      <CreateLogicalBackupConfigStep
        workspaceId="workspace-id"
        initialConfig={enteredConfig}
        onBack={onBack}
        onContinue={onContinue}
      />,
    );
    await screen.findByText('Entered archive');
    fireEvent.click(screen.getAllByRole('checkbox')[0]);

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    const changedSettings = {
      settings: expect.objectContaining({ notificationTriggers: ['BACKUP_SUCCESS'] }),
    };

    expect(onBack).toHaveBeenCalledWith(expect.objectContaining(changedSettings));
    expect(onContinue).toHaveBeenCalledWith(expect.objectContaining(changedSettings));
  });
});
