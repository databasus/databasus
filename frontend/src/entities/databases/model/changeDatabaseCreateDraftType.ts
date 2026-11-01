import { createDefaultDatabaseSettings } from './createDefaultDatabaseSettings';
import type { DatabaseCreateDraft } from './DatabaseConnectionDraft';
import type { DatabaseType } from './DatabaseType';

// Settings entered for one type do not apply to another, and the new type derives its SSL mode
// from the host again.
export const changeDatabaseCreateDraftType = (
  draft: DatabaseCreateDraft,
  type: DatabaseType,
): DatabaseCreateDraft => ({
  ...draft,
  settings: createDefaultDatabaseSettings(type),
  hasUserChosenSslMode: false,
});
