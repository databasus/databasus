import { describe, expect, it } from 'vitest';

import { createDefaultDatabaseSettings } from './createDefaultDatabaseSettings';
import type { DatabaseCreateDraft, DatabaseEditDraft } from './DatabaseConnectionDraft';
import { DatabaseType } from './DatabaseType';
import { toDatabaseSettingsTarget } from './toDatabaseSettingsTarget';

const settings = createDefaultDatabaseSettings(DatabaseType.MYSQL);

const createDraft: DatabaseCreateDraft = {
  scenario: 'create',
  workspaceId: 'workspace-id',
  name: 'Shop',
  notifierIds: [],
  settings,
  hasUserChosenSslMode: false,
};

const editDraft: DatabaseEditDraft = {
  scenario: 'edit',
  workspaceId: 'workspace-id',
  databaseId: 'database-id',
  name: 'Shop',
  notifierIds: [],
  savedSettings: settings,
  settings,
  hasUserChosenSslMode: true,
};

describe('toDatabaseSettingsTarget', () => {
  it('names no database for a new database', () => {
    expect(toDatabaseSettingsTarget(createDraft)).toEqual({
      workspaceId: 'workspace-id',
      databaseId: undefined,
      settings,
    });
  });

  it('names the saved database when editing', () => {
    expect(toDatabaseSettingsTarget(editDraft).databaseId).toBe('database-id');
  });
});
