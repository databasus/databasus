// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type {
  BackupConfig,
  BackupConfigInput,
  StorageSwitch,
} from '../../../../entities/backups/config';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { API_BASE_PATH } from '../../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { EditLogicalBackupConfigComponent } from './EditLogicalBackupConfigComponent';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
};
const config: BackupConfig = {
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: {
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

describe('EditLogicalBackupConfigComponent', () => {
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
});
