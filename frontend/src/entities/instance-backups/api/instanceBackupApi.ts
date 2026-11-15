import type { InstanceBackup } from '@/entities/instance-backups/model/InstanceBackup';
import type { InstanceBackupDestinationChoice } from '@/entities/instance-backups/model/InstanceBackupDestinationChoice';
import type { InstanceBackupSettings } from '@/entities/instance-backups/model/InstanceBackupSettings';
import type { InstanceBackupStatus } from '@/entities/instance-backups/model/InstanceBackupStatus';
import { API_BASE_PATH, type ApiComponents, type ReadOptions } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';
import { getApplicationServer } from '@/shared/config';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';

// The largest page the contract serves.
const DESTINATION_PAGE_LIMIT = 100;

type Schemas = ApiComponents['schemas'];

export const instanceBackupApi = {
  async getSettings(options: ReadOptions = {}): Promise<InstanceBackupSettings> {
    return requireData(await getApiClient(options).GET('/instance-backup-settings'));
  },

  async updateSettings(settings: InstanceBackupSettings): Promise<InstanceBackupSettings> {
    return requireData(await getApiClient().PUT('/instance-backup-settings', { body: settings }));
  },

  async getStatus(options: ReadOptions = {}): Promise<InstanceBackupStatus> {
    return requireData(await getApiClient(options).GET('/instance-backup-statuses'));
  },

  // The picker searches locally, so it needs every choice; an instance has few storages.
  async listAllDestinationChoices(
    options: ReadOptions = {},
  ): Promise<InstanceBackupDestinationChoice[]> {
    return collectCursorPages(
      async (after: string | undefined) =>
        requireData(
          await getApiClient(options).GET('/instance-backup-destinations', {
            params: { query: { limit: DESTINATION_PAGE_LIMIT, after } },
          }),
        ),
      options,
    );
  },

  async listBackups(
    page: { limit: number; after?: string },
    options: ReadOptions = {},
  ): Promise<Schemas['InstanceBackupPage']> {
    return requireData(
      await getApiClient(options).GET('/instance-backups', { params: { query: page } }),
    );
  },

  async startBackup(): Promise<InstanceBackup> {
    return requireData(await getApiClient().POST('/instance-backups'));
  },

  async createDownloadToken(
    instanceBackupId: string,
    copyId: string,
  ): Promise<Schemas['InstanceBackupDownloadToken']> {
    return requireData(
      await getApiClient().POST('/instance-backups/{instanceBackupId}/download-tokens', {
        params: { path: { instanceBackupId } },
        body: { copyId },
      }),
    );
  },

  // A browser link cannot carry the session, so the single-use token is the credential.
  getDownloadUrl(token: string): string {
    const query = new URLSearchParams({ token });

    return `${getApplicationServer()}${API_BASE_PATH}/instance-backup-downloads?${query.toString()}`;
  },

  async updateReminder(isDeferred: boolean): Promise<Schemas['InstanceBackupReminder']> {
    return requireData(
      await getApiClient().PUT('/instance-backup-reminders', { body: { isDeferred } }),
    );
  },
};
