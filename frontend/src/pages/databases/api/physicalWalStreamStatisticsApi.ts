import type { WalStreamStatisticsPeriod } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatisticsPeriod';
import { getApiClient, requireData } from '@/shared/api';

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
