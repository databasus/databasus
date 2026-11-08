import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';

export const diskApi = {
  async getDiskUsage(options: ReadOptions = {}) {
    return requireData(await getApiClient(options).GET('/system/disk-usages'));
  },
};
