import { getApiClient, requireData } from '@/shared/api';

export const healthcheckAttemptApi = {
  async getAttemptsByDatabase(databaseId: string, since: Date, after?: string) {
    return requireData(
      await getApiClient().GET('/healthcheck-attempts', {
        params: { query: { databaseId, since: since.toISOString(), after } },
      }),
    );
  },
};
