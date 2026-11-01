export interface FullBackupRequestDependencies {
  requestFullBackup: () => Promise<unknown>;
  refreshNow: () => void;
  reportRequested: () => void;
  reportError: (error: unknown) => void;
}

// Sends the full backup request and keeps the button busy until the screen can show the result.
// The backend stores the request before it answers, so after a success the refresh started here
// reports the full backup as pending; until that refresh lands the button must not come back. A
// scheduled refresh that lands while the request is still being sent frees nothing either. The
// class holds no React state, so the flow is testable without a DOM.
export class FullBackupRequestController {
  private isRequesting = false;
  private isRequestInFlight = false;

  constructor(
    private readonly dependencies: FullBackupRequestDependencies,
    private readonly onRequestingChange: (isRequesting: boolean) => void,
  ) {}

  async requestFullBackup() {
    if (this.isRequesting) {
      return;
    }

    this.setRequesting(true);
    this.isRequestInFlight = true;

    try {
      await this.dependencies.requestFullBackup();
      this.isRequestInFlight = false;
      this.dependencies.reportRequested();
      this.dependencies.refreshNow();
    } catch (e) {
      this.isRequestInFlight = false;
      this.dependencies.reportError(e);
      this.setRequesting(false);
    }
  }

  handleRefreshLanded() {
    if (!this.isRequestInFlight) {
      this.setRequesting(false);
    }
  }

  // Reports every change, even to the value it already holds: a controller created again after a
  // language switch starts idle, and its first landed refresh must still free a busy button.
  private setRequesting(isRequesting: boolean) {
    this.isRequesting = isRequesting;
    this.onRequestingChange(isRequesting);
  }
}
