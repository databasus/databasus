import { getApiClient, requireData } from '../../../shared/api';
import type { WalStreamStatisticsPeriod } from '../model/wal-stream-monitoring/WalStreamStatisticsPeriod';

export const physicalWalStreamStatisticsApi = {
  async getWalStreamStatistics(databaseId: string, period: WalStreamStatisticsPeriod) {
    return requireData(
      await getApiClient().GET('/wal-streams/{databaseId}/statistics', {
        params: { path: { databaseId }, query: { period } },
      }),
    );
  },
  async getWalStreamLive(databaseId: string) {
    return requireData(
      await getApiClient().GET('/wal-streams/{databaseId}', { params: { path: { databaseId } } }),
    );
  },
};
