import { API_BASE_PATH, getApiClient, requireData, requireSuccess } from '../../../shared/api';
import { getApplicationServer } from '../../../shared/config';
import type { DownloadToken } from '../model/DownloadToken';
import type { LogicalBackup } from '../model/LogicalBackup';
import type { LogicalBackupListFilter } from '../model/LogicalBackupListFilter';
import type { LogicalBackupPage } from '../model/LogicalBackupPage';

interface LogicalBackupPageRequest {
  databaseId: string;
  filter: LogicalBackupListFilter;
  limit: number;
  after?: string;
}

export const logicalBackupApi = {
  async listLogicalBackups({
    databaseId,
    filter,
    limit,
    after,
  }: LogicalBackupPageRequest): Promise<LogicalBackupPage> {
    return requireData(
      await getApiClient().GET('/logical-backups', {
        params: {
          query: {
            databaseId,
            status: filter.statuses,
            createdBefore: filter.createdBefore,
            limit,
            after,
          },
        },
      }),
    );
  },

  // Answers at once with the backup in progress; the list follows it from there.
  async startLogicalBackup(databaseId: string): Promise<LogicalBackup> {
    return requireData(await getApiClient().POST('/logical-backups', { body: { databaseId } }));
  },

  async cancelLogicalBackup(backupId: string): Promise<LogicalBackup> {
    return requireData(
      await getApiClient().POST('/logical-backups/{backupId}/cancel', {
        params: { path: { backupId } },
      }),
    );
  },

  async deleteLogicalBackup(backupId: string): Promise<void> {
    requireSuccess(
      await getApiClient().DELETE('/logical-backups/{backupId}', {
        params: { path: { backupId } },
      }),
    );
  },

  async createDownloadToken(backupId: string): Promise<DownloadToken> {
    return requireData(await getApiClient().POST('/download-tokens', { body: { backupId } }));
  },

  // A browser link cannot carry the session, so the single-use token is the credential of the
  // download.
  getLogicalBackupFileUrl(backupId: string, token: string): string {
    const query = new URLSearchParams({ token });

    return `${getApplicationServer()}${API_BASE_PATH}/logical-backups/${encodeURIComponent(backupId)}/file?${query.toString()}`;
  },
};
