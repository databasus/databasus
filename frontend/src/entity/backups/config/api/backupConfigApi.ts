import { getApiClient, requireData, requireSuccess } from '../../../../shared/api';
import type { BackupConfig } from '../model/BackupConfig';
import type { BackupConfigInput } from '../model/BackupConfigInput';
import type { DatabaseTransfer } from '../model/DatabaseTransfer';

// The largest page the contract serves, so a workspace's configs arrive in as few requests as
// possible.
const BACKUP_CONFIG_PAGE_LIMIT = 100;

export const backupConfigApi = {
  // The database list shows the storage of every card, so this follows the cursor to the last page.
  // A database whose config was never saved is not listed.
  async listAllWorkspaceBackupConfigs(workspaceId: string): Promise<BackupConfig[]> {
    const backupConfigs: BackupConfig[] = [];
    let after: string | undefined;

    do {
      const page = requireData(
        await getApiClient().GET('/backup-configs', {
          params: { query: { workspaceId, limit: BACKUP_CONFIG_PAGE_LIMIT, after } },
        }),
      );

      backupConfigs.push(...page.items);
      after = page.nextCursor;
    } while (after !== undefined);

    return backupConfigs;
  },

  // The first page of the configs that back up into the storage; a small limit answers whether the
  // storage is used, or used by more than one database, without reading every config.
  async listStorageBackupConfigs(storageId: string, limit: number): Promise<BackupConfig[]> {
    const page = requireData(
      await getApiClient().GET('/backup-configs', {
        params: { query: { storageId, limit } },
      }),
    );

    return page.items;
  },

  // A database whose config was never saved reads as the default of its backup method.
  async getBackupConfig(databaseId: string): Promise<BackupConfig> {
    return requireData(
      await getApiClient().GET('/backup-configs/{databaseId}', {
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
    requireSuccess(
      await getApiClient().POST('/databases/{databaseId}/transfer', {
        params: { path: { databaseId } },
        body: transfer,
      }),
    );
  },
};
