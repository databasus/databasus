import type {
  Database,
  DatabaseSettings,
  DatabaseSettingsTarget,
} from '../../../../entity/databases';

// A new database has no ID yet; a saved one names itself, so the server fills in the secrets the
// form left empty.
export const toDatabaseSettingsTarget = (
  database: Database,
  settings: DatabaseSettings,
): DatabaseSettingsTarget => ({
  workspaceId: database.workspaceId,
  databaseId: database.id || undefined,
  settings,
});
