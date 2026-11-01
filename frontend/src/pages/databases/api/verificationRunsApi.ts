import { getApiClient, requireData } from '../../../shared/api';
export const verificationRunsApi = {
  async listByDatabase(databaseId: string, limit?: number, after?: string) {
    return requireData(
      await getApiClient().GET('/verifications', {
        params: { query: { databaseId, limit, after } },
      }),
    );
  },
  async getById(verificationId: string) {
    return requireData(
      await getApiClient().GET('/verifications/{verificationId}', {
        params: { path: { verificationId } },
      }),
    );
  },
  async startVerification(backupId: string) {
    return requireData(await getApiClient().POST('/verifications', { body: { backupId } }));
  },
  async cancel(verificationId: string) {
    return requireData(
      await getApiClient().POST('/verifications/{verificationId}/cancel', {
        params: { path: { verificationId } },
      }),
    );
  },
};
