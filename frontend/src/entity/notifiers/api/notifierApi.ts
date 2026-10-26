import { getApiClient, requireData, requireSuccess } from '../../../shared/api';
import type { Notifier } from '../models/Notifier';
import type { NotifierSettings } from '../models/NotifierSettings';
import type { NotifierTestResult } from '../models/NotifierTestResult';

// The largest page the contract serves, so a workspace's notifiers arrive in as few requests as
// possible.
const NOTIFIER_PAGE_LIMIT = 100;

export const notifierApi = {
  // The notifier screen and the database forms need every notifier of the workspace, so this
  // follows the cursor to the last page.
  async listAllNotifiers(workspaceId: string): Promise<Notifier[]> {
    const notifiers: Notifier[] = [];
    let after: string | undefined;

    do {
      const page = requireData(
        await getApiClient().GET('/notifiers', {
          params: { query: { workspaceId, limit: NOTIFIER_PAGE_LIMIT, after } },
        }),
      );

      notifiers.push(...page.items);
      after = page.nextCursor;
    } while (after !== undefined);

    return notifiers;
  },

  async getNotifier(notifierId: string): Promise<Notifier> {
    return requireData(
      await getApiClient().GET('/notifiers/{notifierId}', {
        params: { path: { notifierId } },
      }),
    );
  },

  async createNotifier(
    workspaceId: string,
    name: string,
    settings: NotifierSettings,
  ): Promise<Notifier> {
    return requireData(
      await getApiClient().POST('/notifiers', {
        body: { workspaceId, name, settings },
      }),
    );
  },

  // A secret the settings leave out keeps its stored value.
  async updateNotifier(
    notifierId: string,
    name: string,
    settings: NotifierSettings,
  ): Promise<Notifier> {
    return requireData(
      await getApiClient().PUT('/notifiers/{notifierId}', {
        params: { path: { notifierId } },
        body: { name, settings },
      }),
    );
  },

  async deleteNotifier(notifierId: string): Promise<void> {
    requireSuccess(
      await getApiClient().DELETE('/notifiers/{notifierId}', {
        params: { path: { notifierId } },
      }),
    );
  },

  async testNotifier(notifierId: string): Promise<NotifierTestResult> {
    return requireData(
      await getApiClient().POST('/notifiers/{notifierId}/test', {
        params: { path: { notifierId } },
      }),
    );
  },

  // A secret the settings leave out is taken from the saved notifier the test names.
  async testNotifierSettings(
    workspaceId: string,
    settings: NotifierSettings,
    notifierId?: string,
  ): Promise<NotifierTestResult> {
    return requireData(
      await getApiClient().POST('/notifier-tests', {
        body: { workspaceId, notifierId, settings },
      }),
    );
  },

  async transferNotifier(notifierId: string, targetWorkspaceId: string): Promise<void> {
    requireSuccess(
      await getApiClient().POST('/notifiers/{notifierId}/transfer', {
        params: { path: { notifierId } },
        body: { targetWorkspaceId },
      }),
    );
  },
};
