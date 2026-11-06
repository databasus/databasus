import type {
  DatabaseCreateDraft,
  DatabaseEditDraft,
} from '@/entities/databases/model/DatabaseConnectionDraft';
import type { DatabaseSettingsTarget } from '@/entities/databases/model/DatabaseSettingsTarget';

// A saved database names itself, so the server fills in the secrets the form left empty. A restore
// target draft is not accepted, so it cannot produce a request that reaches a stored secret.
export const toDatabaseSettingsTarget = (
  draft: DatabaseCreateDraft | DatabaseEditDraft,
): DatabaseSettingsTarget => ({
  workspaceId: draft.workspaceId,
  databaseId: draft.scenario === 'edit' ? draft.databaseId : undefined,
  settings: draft.settings,
});
