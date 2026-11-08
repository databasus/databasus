import type { WalStreamStatisticsPeriod } from '@/pages/databases/wal-monitoring/model/WalStreamStatisticsPeriod';
import { type ReadOptions, getApiClient, requireData } from '@/shared/api';

export const physicalWalStreamStatisticsApi = {
  async getWalStreamStatistics(
    databaseId: string,
    period: WalStreamStatisticsPeriod,
    options: ReadOptions = {},
  ) {
    return requireData(
      await getApiClient(options).GET('/wal-streams/{databaseId}/statistics', {
        params: { path: { databaseId }, query: { period } },
      }),
    );
  },
  async getWalStreamLive(databaseId: string, options: ReadOptions = {}) {
    return requireData(
      await getApiClient(options).GET('/wal-streams/{databaseId}', {
        params: { path: { databaseId } },
      }),
    );
  },
};
