import type {
  PageVisibility,
  RefreshTimer,
} from '@/pages/databases/wal-monitoring/lib/walStreamRefreshController';
import type {
  WalStreamLive,
  WalStreamLiveResponse,
} from '@/pages/databases/wal-monitoring/model/walStreamContract';

// The running streamer answers from memory, so a reading every second is cheap and makes the
// buffer and the last save move in about real time.
export const WAL_STREAM_LIVE_INTERVAL_MS = 1_000;

interface WalStreamLiveDependencies {
  fetchLive: () => Promise<WalStreamLiveResponse>;
  timer: RefreshTimer;
  pageVisibility: PageVisibility;
}

// Reads the current values on start, every second and at once when the browser tab becomes
// visible again, never while it is hidden, and never while the previous reading is still on its
// way, so a slow server does not pile requests up. A failed reading keeps the last values: the
// statistics, refreshed on their own schedule, stay on screen underneath and say when they fail.
export class WalStreamLiveController {
  private isReading = false;
  private isStopped = false;
  private intervalHandle: unknown = null;
  private unsubscribeFromVisibility: (() => void) | null = null;

  constructor(
    private readonly dependencies: WalStreamLiveDependencies,
    private readonly onLiveChange: (live: WalStreamLive | null) => void,
  ) {}

  start() {
    this.isStopped = false;

    this.unsubscribeFromVisibility = this.dependencies.pageVisibility.subscribe(() => {
      if (this.dependencies.pageVisibility.isVisible()) {
        void this.read();
      }
    });

    this.intervalHandle = this.dependencies.timer.setInterval(() => {
      if (this.dependencies.pageVisibility.isVisible()) {
        void this.read();
      }
    }, WAL_STREAM_LIVE_INTERVAL_MS);

    void this.read();
  }

  stop() {
    this.isStopped = true;

    if (this.intervalHandle !== null) {
      this.dependencies.timer.clearInterval(this.intervalHandle);
      this.intervalHandle = null;
    }

    this.unsubscribeFromVisibility?.();
    this.unsubscribeFromVisibility = null;
  }

  private async read() {
    if (this.isReading) {
      return;
    }

    this.isReading = true;

    try {
      const response = await this.dependencies.fetchLive();

      if (!this.isStopped) {
        this.onLiveChange(response.isStreamerRunning ? response.current : null);
      }
    } catch {
      // The last values stay; the next tick tries again.
    } finally {
      this.isReading = false;
    }
  }
}
