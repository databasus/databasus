import { getApiClient, requireData } from '../../../shared/api';
import type { HealthcheckConfig } from '../model/HealthcheckConfig';

export const healthcheckConfigApi = {
  async saveHealthcheckConfig(config: HealthcheckConfig) {
    const { databaseId, ...input } = config;
    return requireData(
      await getApiClient().PUT('/healthcheck-configs/{databaseId}', {
        params: { path: { databaseId } },
        body: input,
      }),
    );
  },
  async getHealthcheckConfig(databaseId: string) {
    return requireData(
      await getApiClient().GET('/healthcheck-configs/{databaseId}', {
        params: { path: { databaseId } },
      }),
    );
  },
};
