import type { WalStreamStatisticsPeriod } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatisticsPeriod';
import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';

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
