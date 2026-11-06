import type { Database } from '@/entities/databases/model/Database';
import type { DatabaseCreate } from '@/entities/databases/model/DatabaseCreate';
import type { DatabaseSettingsTarget } from '@/entities/databases/model/DatabaseSettingsTarget';
import type { DatabaseTestResult } from '@/entities/databases/model/DatabaseTestResult';
import type { DatabaseUpdate } from '@/entities/databases/model/DatabaseUpdate';
import type { LimitedUser, LimitedUserCheck } from '@/entities/databases/model/LimitedUser';
import { toDatabaseSettingsRequest } from '@/entities/databases/model/toDatabaseSettingsRequest';
import { getApiClient, requireData, requireSuccess } from '@/shared/api';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';

// The largest page the contract serves, so a workspace's databases arrive in as few requests as
// possible.
const DATABASE_PAGE_LIMIT = 100;

const toSettingsTargetRequest = (target: DatabaseSettingsTarget): DatabaseSettingsTarget => ({
  workspaceId: target.workspaceId,
  databaseId: target.databaseId,
  settings: toDatabaseSettingsRequest(target.settings),
});

// Narrows the list to the databases that send to one notifier.
interface DatabaseListFilter {
  notifierId?: string;
}

export const databaseApi = {
  // The database screen and the transfer dialogs need every database of the workspace, or every
  // database that sends to one notifier, so this follows the cursor to the last page.
  async listAllDatabases(
    workspaceId: string,
    { notifierId }: DatabaseListFilter = {},
  ): Promise<Database[]> {
    return collectCursorPages(async (after: string | undefined) =>
      requireData(
        await getApiClient().GET('/databases', {
          params: { query: { workspaceId, notifierId, limit: DATABASE_PAGE_LIMIT, after } },
        }),
      ),
    );
  },

  async getDatabase(databaseId: string): Promise<Database> {
    return requireData(
      await getApiClient().GET('/databases/{databaseId}', {
        params: { path: { databaseId } },
      }),
    );
  },

  async createDatabase(databaseCreate: DatabaseCreate): Promise<Database> {
    return requireData(
      await getApiClient().POST('/databases', {
        body: {
          workspaceId: databaseCreate.workspaceId,
          name: databaseCreate.name,
          settings: toDatabaseSettingsRequest(databaseCreate.settings),
          notifierIds: databaseCreate.notifierIds,
        },
      }),
    );
  },

  // A secret the settings leave empty keeps its stored value while the resource address stays the
  // stored one. Only the editable fields are sent, so
  // a whole Database can be passed.
  async updateDatabase(databaseId: string, databaseUpdate: DatabaseUpdate): Promise<Database> {
    return requireData(
      await getApiClient().PUT('/databases/{databaseId}', {
        params: { path: { databaseId } },
        body: {
          name: databaseUpdate.name,
          settings: toDatabaseSettingsRequest(databaseUpdate.settings),
          notifierIds: databaseUpdate.notifierIds,
        },
      }),
    );
  },

  async deleteDatabase(databaseId: string): Promise<void> {
    requireSuccess(
      await getApiClient().DELETE('/databases/{databaseId}', {
        params: { path: { databaseId } },
      }),
    );
  },

  async copyDatabase(databaseId: string): Promise<Database> {
    return requireData(
      await getApiClient().POST('/databases/{databaseId}/copy', {
        params: { path: { databaseId } },
      }),
    );
  },

  async testDatabase(databaseId: string): Promise<DatabaseTestResult> {
    return requireData(
      await getApiClient().POST('/databases/{databaseId}/test', {
        params: { path: { databaseId } },
      }),
    );
  },

  // A secret the settings leave empty is taken from the saved database the target names.
  async testDatabaseSettings(target: DatabaseSettingsTarget): Promise<DatabaseTestResult> {
    return requireData(
      await getApiClient().POST('/database-tests', { body: toSettingsTargetRequest(target) }),
    );
  },

  async checkLimitedUser(target: DatabaseSettingsTarget): Promise<LimitedUserCheck> {
    return requireData(
      await getApiClient().POST('/limited-user-checks', { body: toSettingsTargetRequest(target) }),
    );
  },

  async createLimitedUser(target: DatabaseSettingsTarget): Promise<LimitedUser> {
    return requireData(
      await getApiClient().POST('/limited-users', { body: toSettingsTargetRequest(target) }),
    );
  },
};
