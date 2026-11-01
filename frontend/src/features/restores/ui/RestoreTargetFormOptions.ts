import type { RestoreTargetDraft } from '../../../entities/databases';
import type { PostgresqlRestoreOptions } from '../../../entities/restores';

// What the page's connection form needs to edit a target. The form cannot be imported here, because
// features do not import each other, so the page passes it in.
export interface RestoreTargetFormOptions {
  draft: RestoreTargetDraft;
  onDraftChange: (draft: RestoreTargetDraft) => void;
  restoreOptions: PostgresqlRestoreOptions;
  onRestoreOptionsChange: (restoreOptions: PostgresqlRestoreOptions) => void;
  submitLabel: string;
  onSubmit: (draft: RestoreTargetDraft) => Promise<void>;
}
