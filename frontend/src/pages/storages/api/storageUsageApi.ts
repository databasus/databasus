import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';

export const storageUsageApi = {
  async getUsage(storageId: string, options: ReadOptions = {}) {
    return requireData(
      await getApiClient(options).GET('/storages/{storageId}/backup-usage', {
        params: { path: { storageId } },
      }),
    );
  },
};
