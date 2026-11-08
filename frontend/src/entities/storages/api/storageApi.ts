import type { Storage } from '@/entities/storages/model/Storage';
import type { StorageSettings } from '@/entities/storages/model/StorageSettings';
import type { StorageTestResult } from '@/entities/storages/model/StorageTestResult';
import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData, requireSuccess } from '@/shared/api';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';

// The largest page the contract serves, so a workspace's storages arrive in as few requests as
// possible.
const STORAGE_PAGE_LIMIT = 100;

export const storageApi = {
  // The storage screen and the backup config forms need every storage of the workspace, so this
  // follows the cursor to the last page.
  async listAllStorages(workspaceId: string, options: ReadOptions = {}): Promise<Storage[]> {
    return collectCursorPages(
      async (after: string | undefined) =>
        requireData(
          await getApiClient(options).GET('/storages', {
            params: { query: { workspaceId, limit: STORAGE_PAGE_LIMIT, after } },
          }),
        ),
      options,
    );
  },

  async getStorage(storageId: string, options: ReadOptions = {}): Promise<Storage> {
    return requireData(
      await getApiClient(options).GET('/storages/{storageId}', {
        params: { path: { storageId } },
      }),
    );
  },

  async createStorage(
    workspaceId: string,
    name: string,
    settings: StorageSettings,
  ): Promise<Storage> {
    return requireData(
      await getApiClient().POST('/storages', {
        body: { workspaceId, name, settings },
      }),
    );
  },

  // Omitted secrets remain stored while the resource address stays the stored one; SFTP removes
  // credentials excluded by authMethods.
  async updateStorage(
    storageId: string,
    name: string,
    settings: StorageSettings,
  ): Promise<Storage> {
    return requireData(
      await getApiClient().PUT('/storages/{storageId}', {
        params: { path: { storageId } },
        body: { name, settings },
      }),
    );
  },

  async deleteStorage(storageId: string, shouldAbandonCleanup = false): Promise<void> {
    requireSuccess(
      await getApiClient().DELETE('/storages/{storageId}', {
        params: {
          path: { storageId },
          query: shouldAbandonCleanup ? { shouldAbandonCleanup: true } : undefined,
        },
      }),
    );
  },

  async testStorage(storageId: string): Promise<StorageTestResult> {
    return requireData(
      await getApiClient().POST('/storages/{storageId}/test', {
        params: { path: { storageId } },
      }),
    );
  },

  // Omitted selected credentials come from the named storage, using the same policy as saving.
  async testStorageSettings(
    workspaceId: string,
    settings: StorageSettings,
    storageId?: string,
  ): Promise<StorageTestResult> {
    return requireData(
      await getApiClient().POST('/storage-tests', {
        body: { workspaceId, storageId, settings },
      }),
    );
  },

  async transferStorage(storageId: string, targetWorkspaceId: string): Promise<void> {
    requireSuccess(
      await getApiClient().POST('/storages/{storageId}/transfer', {
        params: { path: { storageId } },
        body: { targetWorkspaceId },
      }),
    );
  },
};
