import { getApiClient, requireData } from '../../../shared/api';

export const diskApi = {
  async getDiskUsage() {
    return requireData(await getApiClient().GET('/system/disk-usages'));
  },
};
