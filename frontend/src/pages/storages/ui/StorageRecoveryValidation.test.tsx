// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import type { Storage } from '@/entities/storages';
import { StorageComponent } from '@/pages/storages/ui/StorageComponent';
import { API_BASE_PATH } from '@/shared/api';
import { DICTIONARIES } from '@/shared/i18n';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const text = DICTIONARIES.en.storages.recoveryValidation;

const recoveredStorage: Storage = {
  id: 'recovered-storage',
  workspaceId: 'workspace-id',
  recoveryValidationStatus: 'REQUIRES_VALIDATION',
  name: 'Recovered local backups',
  settings: { type: 'LOCAL' },
};

// The storage read answers with the status the server holds at that moment, so a test can model a
// probe that validated it and one that did not.
const serveStorage = (statusAfterTest: Storage['recoveryValidationStatus']) => {
  let isTested = false;
  server.use(
    mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 0 }),
    http.get(`*${API_BASE_PATH}/storages/:storageId`, () =>
      HttpResponse.json({
        ...recoveredStorage,
        recoveryValidationStatus: isTested ? statusAfterTest : 'REQUIRES_VALIDATION',
      }),
    ),
    http.post(`*${API_BASE_PATH}/storages/:storageId/test`, () => {
      isTested = true;
      return HttpResponse.json({ isConnected: true });
    }),
  );
};

const renderStorage = (isCanManageStorages: boolean) =>
  renderScreen(
    <StorageComponent
      storageId={recoveredStorage.id}
      onStorageChanged={() => {}}
      onStorageDeleted={() => {}}
      onStorageTransferred={() => {}}
      isCanManageStorages={isCanManageStorages}
    />,
  );

describe('Recovered local storage validation', () => {
  it('explains that an editor validates it by testing the connection', async () => {
    serveStorage('REQUIRES_VALIDATION');
    renderStorage(true);

    expect(await screen.findByText(text.title)).toBeTruthy();
    expect(screen.getByText(text.editorAction)).toBeTruthy();
  });

  it('reports the storage available after an editor validates it', async () => {
    serveStorage('AVAILABLE');
    const { user } = renderStorage(true);

    await user.click(
      await screen.findByRole('button', { name: DICTIONARIES.en.storages.actions.testConnection }),
    );

    expect(await screen.findByText(text.validated)).toBeTruthy();
    expect(screen.queryByText(text.title)).toBeNull();
  });

  it('keeps the storage unavailable when an editor test did not validate it', async () => {
    serveStorage('REQUIRES_VALIDATION');
    const { user } = renderStorage(true);

    await user.click(
      await screen.findByRole('button', { name: DICTIONARIES.en.storages.actions.testConnection }),
    );

    expect(await screen.findByText(text.notValidated)).toBeTruthy();
    expect(screen.getByText(text.title)).toBeTruthy();
  });

  it('explains why a viewer successful test leaves the storage unavailable', async () => {
    serveStorage('REQUIRES_VALIDATION');
    const { user } = renderStorage(false);

    expect(await screen.findByText(text.viewerAction)).toBeTruthy();

    await user.click(
      screen.getByRole('button', { name: DICTIONARIES.en.storages.actions.testConnection }),
    );

    expect(await screen.findByText(text.viewerConnected)).toBeTruthy();
    expect(screen.getByText(text.title)).toBeTruthy();
  });
});
