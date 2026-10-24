import { getApiClient, requireData } from '../../../shared/api';

export const systemApi = {
  async getVersion() {
    return requireData(await getApiClient().GET('/system/version', { cache: 'no-store' }));
  },
};
