import { createDefaultDatabaseSettings } from '@/entities/databases/model/createDefaultDatabaseSettings';
import type { DatabaseCreateDraft } from '@/entities/databases/model/DatabaseConnectionDraft';
import type { DatabaseType } from '@/entities/databases/model/DatabaseType';

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
