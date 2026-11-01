import { useState } from 'react';

import type {
  DatabaseCreateDraft,
  DatabaseEditDraft,
  DatabaseSettings,
  RestoreTargetDraft,
  RestoreTargetSettings,
} from '@/entities/databases';
import type { PostgresqlRestoreOptions } from '@/entities/restores';
import { renderScreen } from '@/shared/lib/test/renderScreen';

import { DatabaseConnectionForm } from './DatabaseConnectionForm';

export const NO_RESTORE_OPTIONS: PostgresqlRestoreOptions = {
  isExcludeExtensions: false,
  isRestoreOwnership: false,
  isRestorePrivileges: false,
};

export const buildCreateDraft = (settings: DatabaseSettings): DatabaseCreateDraft => ({
  scenario: 'create',
  workspaceId: 'workspace-id',
  name: 'Shop',
  notifierIds: [],
  settings,
  hasUserChosenSslMode: false,
});

export const buildEditDraft = (settings: DatabaseSettings): DatabaseEditDraft => ({
  scenario: 'edit',
  workspaceId: 'workspace-id',
  databaseId: 'database-id',
  name: 'Shop',
  notifierIds: [],
  savedSettings: settings,
  settings,
  hasUserChosenSslMode: true,
});

export const buildRestoreDraft = (settings: RestoreTargetSettings): RestoreTargetDraft => ({
  scenario: 'restore',
  settings,
  hasUserChosenSslMode: false,
});

// eslint-disable-next-line react/only-export-components -- a test fixture has no fast refresh
const ControlledEditableForm = ({
  initialDraft,
  onSubmit,
}: {
  initialDraft: DatabaseCreateDraft | DatabaseEditDraft;
  onSubmit: (draft: DatabaseCreateDraft | DatabaseEditDraft) => Promise<void> | void;
}) => {
  const [draft, setDraft] = useState(initialDraft);

  return draft.scenario === 'create' ? (
    <DatabaseConnectionForm
      draft={draft}
      onDraftChange={setDraft}
      onSubmit={onSubmit}
      submitLabel="Submit draft"
    />
  ) : (
    <DatabaseConnectionForm
      draft={draft}
      onDraftChange={setDraft}
      onSubmit={onSubmit}
      submitLabel="Submit draft"
    />
  );
};

// eslint-disable-next-line react/only-export-components -- a test fixture has no fast refresh
const ControlledRestoreForm = ({
  initialDraft,
  restoreOptions: initialRestoreOptions,
}: {
  initialDraft: RestoreTargetDraft;
  restoreOptions: PostgresqlRestoreOptions;
}) => {
  const [draft, setDraft] = useState(initialDraft);
  const [restoreOptions, setRestoreOptions] = useState(initialRestoreOptions);

  return (
    <DatabaseConnectionForm
      draft={draft}
      onDraftChange={setDraft}
      restoreOptions={restoreOptions}
      onRestoreOptionsChange={setRestoreOptions}
      onSubmit={() => {}}
      submitLabel="Submit draft"
    />
  );
};

export const renderEditableForm = (
  draft: DatabaseCreateDraft | DatabaseEditDraft,
  onSubmit: (draft: DatabaseCreateDraft | DatabaseEditDraft) => Promise<void> | void = () => {},
) => renderScreen(<ControlledEditableForm initialDraft={draft} onSubmit={onSubmit} />);

export const renderRestoreForm = (
  draft: RestoreTargetDraft,
  restoreOptions: PostgresqlRestoreOptions = NO_RESTORE_OPTIONS,
) => renderScreen(<ControlledRestoreForm initialDraft={draft} restoreOptions={restoreOptions} />);
