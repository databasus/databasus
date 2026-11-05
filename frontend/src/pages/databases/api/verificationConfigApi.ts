import { getApiClient, requireData, type ApiComponents } from '../../../shared/api';
export const verificationConfigApi = {
  async getByDatabaseId(databaseId: string) {
    return requireData(
      await getApiClient().GET('/verification-configs/{databaseId}', {
        params: { path: { databaseId } },
      }),
    );
  },
  async save(databaseId: string, config: ApiComponents['schemas']['VerificationConfigInput']) {
    return requireData(
      await getApiClient().PUT('/verification-configs/{databaseId}', {
        params: { path: { databaseId } },
        body: config,
      }),
    );
  },
};
