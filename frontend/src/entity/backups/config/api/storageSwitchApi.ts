import { getApiClient, requireData } from '../../../../shared/api';

export const storageSwitchApi = {
  async getStorageSwitch(storageSwitchId: string) {
    return requireData(
      await getApiClient().GET('/storage-switches/{storageSwitchId}', {
        params: { path: { storageSwitchId } },
      }),
    );
  },

  async listStorageSwitches(query: { workspaceId: string; databaseId?: string; after?: string }) {
    return requireData(
      await getApiClient().GET('/storage-switches', {
        params: { query },
      }),
    );
  },
};
