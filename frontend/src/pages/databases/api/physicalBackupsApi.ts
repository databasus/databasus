import {
  API_BASE_PATH,
  getApiClient,
  requireData,
  requireSuccess,
  type ApiComponents,
} from '../../../shared/api';
import { getApplicationServer } from '../../../shared/config';
import type { PhysicalBackupsFilters } from '../model/physical-backups/PhysicalBackupsFilters';

export type TriggerPhysicalBackupType = ApiComponents['schemas']['PhysicalBackupStartKind'];
export type PhysicalRestoreTokenCreate = ApiComponents['schemas']['PhysicalRestoreTokenCreate'];

export const physicalBackupsApi = {
  async getPhysicalBackups(
    databaseId: string,
    limit?: number,
    after?: string,
    filters?: PhysicalBackupsFilters,
  ) {
    return requireData(
      await getApiClient().GET('/physical-backups', {
        params: {
          query: {
            databaseId,
            limit,
            after,
            type: filters?.types,
            status: filters?.statuses,
            createdBefore: filters?.createdBefore,
          },
        },
      }),
    );
  },
  async triggerPhysicalBackup(databaseId: string, kind: TriggerPhysicalBackupType) {
    requireSuccess(await getApiClient().POST('/physical-backups', { body: { databaseId, kind } }));
  },
  async cancelPhysicalBackup(backupId: string) {
    requireSuccess(
      await getApiClient().POST('/physical-backups/{backupId}/cancel', {
        params: { path: { backupId } },
      }),
    );
  },
  async deletePhysicalBackup(backupId: string) {
    requireSuccess(
      await getApiClient().DELETE('/physical-backups/{backupId}', {
        params: { path: { backupId } },
      }),
    );
  },
  async createPhysicalRestoreToken(body: PhysicalRestoreTokenCreate) {
    return requireData(await getApiClient().POST('/physical-restore-tokens', { body }));
  },
  getPhysicalRestoreSetUrl(token: string): string {
    return `${getApplicationServer()}${API_BASE_PATH}/physical-restore-sets?${new URLSearchParams({ token })}`;
  },
  getPhysicalRecoveryScriptUrl(): string {
    return `${getApplicationServer()}${API_BASE_PATH}/physical-recovery-scripts`;
  },
};
