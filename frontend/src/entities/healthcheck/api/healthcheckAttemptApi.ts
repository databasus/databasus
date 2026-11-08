import { getApiClient, requireData, type ReadOptions } from '@/shared/api';

export const healthcheckAttemptApi = {
  async getAttemptsByDatabase({
    databaseId,
    since,
    after,
    signal,
  }: ReadOptions & { databaseId: string; since: Date; after?: string }) {
    return requireData(
      await getApiClient({ signal }).GET('/healthcheck-attempts', {
        params: { query: { databaseId, since: since.toISOString(), after } },
      }),
    );
  },
};
