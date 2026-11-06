import type { WalStreamStatistics } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatistics';

export interface WalStreamRefreshState {
  // The newest successful response; a failed refresh never replaces it.
  statistics: WalStreamStatistics | null;
  lastSuccessfulRefreshAt: Date | null;

  // The first load failed, so there is nothing to show but the error and a retry.
  firstLoadError: unknown;

  isLastRefreshFailed: boolean;
}
