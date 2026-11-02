export interface FullBackupRequestDependencies {
  requestFullBackup: () => Promise<unknown>;
  refreshNow: () => void;
  reportRequested: () => void;
  reportError: (error: unknown) => void;
}

// Sends the full backup request and keeps the button busy until the screen can show the result.
// The backend stores the request before it answers, so after a success the refresh started here
// reports the full backup as pending; until that refresh lands the button must not come back. A
// scheduled refresh that lands while the request is still being sent frees nothing either. After
// stopReporting, the request still ends on the server, but its answer reaches nothing: the screen
// that asked is gone. The class holds no React state, so the flow is testable without a DOM.
export class FullBackupRequestController {
  private isRequesting = false;
  private isRequestInFlight = false;
  private isReporting = true;

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

      if (!this.isReporting) return;

      this.dependencies.reportRequested();
      this.dependencies.refreshNow();
    } catch (e) {
      this.isRequestInFlight = false;

      if (!this.isReporting) return;

      this.dependencies.reportError(e);
      this.setRequesting(false);
    }
  }

  handleRefreshLanded() {
    if (this.isRequesting && !this.isRequestInFlight) {
      this.setRequesting(false);
    }
  }

  stopReporting() {
    this.isReporting = false;
  }

  private setRequesting(isRequesting: boolean) {
    this.isRequesting = isRequesting;
    this.onRequestingChange(isRequesting);
  }
}
