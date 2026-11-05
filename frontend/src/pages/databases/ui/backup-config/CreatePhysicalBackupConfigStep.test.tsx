// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { buildPhysicalBackupConfigDraft } from '../../model/physical-backups/buildPhysicalBackupConfigDraft.fixture';
import { CreatePhysicalBackupConfigStep } from './CreatePhysicalBackupConfigStep';

const mockServer = setUpMockApi();
const archiveStorage = {
  id: 'archive-id',
  workspaceId: 'workspace-id',
  name: 'Archive',
  settings: { type: 'LOCAL' as const },
};

const declareStorages = () =>
  mockServer.use(mockApi.get('/storages', 200, { items: [archiveStorage] }));

const chooseArchiveStorage = async () => {
  const storageLabel = await screen.findByText('Storage');

  fireEvent.mouseDown(storageLabel.parentElement!.querySelector('.ant-select-selector')!);
  const option = await waitFor(() => {
    const storageOption = document.querySelector('.ant-select-item-option[title="Archive"]');
    expect(storageOption).toBeTruthy();
    return storageOption!;
  });

  fireEvent.click(option);
};

describe('CreatePhysicalBackupConfigStep', () => {
  it('requires a storage before Continue', async () => {
    declareStorages();
    const onContinue = vi.fn();
    renderScreen(
      <CreatePhysicalBackupConfigStep
        workspaceId="workspace-id"
        backupType="FULL"
        onContinue={onContinue}
      />,
    );
    const continueButton = screen.getByRole('button', { name: 'Continue' });

    expect(continueButton).toHaveProperty('disabled', true);

    await chooseArchiveStorage();
    fireEvent.click(continueButton);

    expect(onContinue).toHaveBeenCalledWith(
      expect.objectContaining({ storageId: archiveStorage.id }),
    );
  });

  it('starts from the given draft instead of the defaults', async () => {
    declareStorages();
    const initialDraft = buildPhysicalBackupConfigDraft({
      config: { storageId: archiveStorage.id },
      retention: { chainsCount: 11 },
    });
    const onContinue = vi.fn();
    renderScreen(
      <CreatePhysicalBackupConfigStep
        workspaceId="workspace-id"
        backupType="FULL_INCREMENTAL"
        initialDraft={initialDraft}
        onContinue={onContinue}
      />,
    );

    expect(screen.getByRole('spinbutton', { name: 'Chains' })).toHaveProperty('value', '11');

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    expect(onContinue).toHaveBeenCalledWith(initialDraft);
  });

  it('gives Back the entered draft even when the step is not complete', async () => {
    declareStorages();
    const onBack = vi.fn();
    const view = renderScreen(
      <CreatePhysicalBackupConfigStep
        workspaceId="workspace-id"
        backupType="FULL_INCREMENTAL"
        onBack={onBack}
        onContinue={() => {}}
      />,
    );

    fireEvent.change(screen.getByRole('spinbutton', { name: 'Chains' }), {
      target: { value: '9' },
    });
    await view.user.click(screen.getByRole('button', { name: 'Back' }));

    expect(onBack).toHaveBeenCalledTimes(1);
    expect(onBack.mock.calls[0][0].settings.retention.chainsCount).toBe(9);
    expect(onBack.mock.calls[0][0].storageId).toBeNull();
  });

  it('offers no Back when the wizard gives no way back', () => {
    declareStorages();
    renderScreen(
      <CreatePhysicalBackupConfigStep
        workspaceId="workspace-id"
        backupType="FULL"
        onContinue={() => {}}
      />,
    );

    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull();
  });
});
