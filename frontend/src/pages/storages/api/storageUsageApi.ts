import { getApiClient, requireData } from '@/shared/api';

export const storageUsageApi = {
  async getUsage(storageId: string) {
    return requireData(
      await getApiClient().GET('/storages/{storageId}/backup-usage', {
        params: { path: { storageId } },
      }),
    );
  },
};
