import type { HealthcheckConfig } from '@/entities/healthcheck/model/HealthcheckConfig';
import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';

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
  async getHealthcheckConfig(databaseId: string, options: ReadOptions = {}) {
    return requireData(
      await getApiClient(options).GET('/healthcheck-configs/{databaseId}', {
        params: { path: { databaseId } },
      }),
    );
  },
};
