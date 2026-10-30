import { getApiClient, requireData, requireSuccess } from '../../../shared/api';
import { collectCursorPages } from '../../../shared/lib/collectCursorPages';
import type { Storage } from '../models/Storage';
import type { StorageSettings } from '../models/StorageSettings';
import type { StorageTestResult } from '../models/StorageTestResult';

// The largest page the contract serves, so a workspace's storages arrive in as few requests as
// possible.
const STORAGE_PAGE_LIMIT = 100;

export const storageApi = {
  // The storage screen and the backup config forms need every storage of the workspace, so this
  // follows the cursor to the last page.
  async listAllStorages(workspaceId: string): Promise<Storage[]> {
    return collectCursorPages(async (after: string | undefined) =>
      requireData(
        await getApiClient().GET('/storages', {
          params: { query: { workspaceId, limit: STORAGE_PAGE_LIMIT, after } },
        }),
      ),
    );
  },

  async getStorage(storageId: string): Promise<Storage> {
    return requireData(
      await getApiClient().GET('/storages/{storageId}', {
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

  // A secret the settings leave out keeps its stored value.
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

  async deleteStorage(storageId: string): Promise<void> {
    requireSuccess(
      await getApiClient().DELETE('/storages/{storageId}', {
        params: { path: { storageId } },
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

  // A secret the settings leave out is taken from the saved storage the test names.
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
