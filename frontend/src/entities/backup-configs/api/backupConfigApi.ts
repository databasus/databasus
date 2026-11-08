import type { BackupConfig } from '@/entities/backup-configs/model/BackupConfig';
import type { BackupConfigInput } from '@/entities/backup-configs/model/BackupConfigInput';
import type { DatabaseTransfer } from '@/entities/backup-configs/model/DatabaseTransfer';
import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData, requireSuccess } from '@/shared/api';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';

// The largest page the contract serves, so a workspace's configs arrive in as few requests as
// possible.
const BACKUP_CONFIG_PAGE_LIMIT = 100;

export const backupConfigApi = {
  // The database list shows the storage of every card, so this follows the cursor to the last page.
  // A database whose config was never saved is not listed.
  async listAllWorkspaceBackupConfigs(
    workspaceId: string,
    options: ReadOptions = {},
  ): Promise<BackupConfig[]> {
    return collectCursorPages(
      async (after: string | undefined) =>
        requireData(
          await getApiClient(options).GET('/backup-configs', {
            params: { query: { workspaceId, limit: BACKUP_CONFIG_PAGE_LIMIT, after } },
          }),
        ),
      options,
    );
  },

  // The first page of the configs that back up into the storage; a small limit answers whether the
  // storage is used, or used by more than one database, without reading every config.
  async listStorageBackupConfigs(
    storageId: string,
    limit: number,
    options: ReadOptions = {},
  ): Promise<BackupConfig[]> {
    const page = requireData(
      await getApiClient(options).GET('/backup-configs', {
        params: { query: { storageId, limit } },
      }),
    );

    return page.items;
  },

  // A database whose config was never saved reads as the default of its backup method.
  async getBackupConfig(databaseId: string, options: ReadOptions = {}): Promise<BackupConfig> {
    return requireData(
      await getApiClient(options).GET('/backup-configs/{databaseId}', {
        params: { path: { databaseId } },
      }),
    );
  },

  async updateBackupConfig(databaseId: string, input: BackupConfigInput): Promise<BackupConfig> {
    return requireData(
      await getApiClient().PUT('/backup-configs/{databaseId}', {
        params: { path: { databaseId } },
        body: input,
      }),
    );
  },

  async transferDatabase(databaseId: string, transfer: DatabaseTransfer): Promise<void> {
    const response = await getApiClient().POST('/databases/{databaseId}/transfer', {
      params: { path: { databaseId } },
      body: transfer,
    });
    requireSuccess(response);
  },

  async hasBackupHistory(
    databaseId: string,
    isPhysical: boolean,
    options: ReadOptions = {},
  ): Promise<boolean> {
    if (isPhysical) {
      const page = requireData(
        await getApiClient(options).GET('/physical-backups', {
          params: { query: { databaseId, limit: 1 } },
        }),
      );
      return page.items.length > 0;
    }
    const page = requireData(
      await getApiClient(options).GET('/logical-backups', {
        params: { query: { databaseId, limit: 1 } },
      }),
    );
    return page.items.length > 0;
  },
};
