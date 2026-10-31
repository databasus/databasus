import { type Database, createDefaultDatabaseSettings } from '../../../entities/databases';

// The target is usually another server, so it starts as a new database of the backup's engine.
// An empty ID keeps the form from treating any stored secret of the source as its own.
export const createRestoreTargetDraft = (sourceDatabase: Database): Database => ({
  id: '',
  name: '',
  workspaceId: sourceDatabase.workspaceId,
  settings: createDefaultDatabaseSettings(sourceDatabase.settings.type),
  notifierIds: [],
});
