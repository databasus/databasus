import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';

export const systemApi = {
  async getVersion(options: ReadOptions = {}) {
    return requireData(await getApiClient(options).GET('/system/version', { cache: 'no-store' }));
  },
};
