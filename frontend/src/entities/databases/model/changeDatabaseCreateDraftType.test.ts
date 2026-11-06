import { describe, expect, it } from 'vitest';

import { changeDatabaseCreateDraftType } from '@/entities/databases/model/changeDatabaseCreateDraftType';
import { createDefaultDatabaseSettings } from '@/entities/databases/model/createDefaultDatabaseSettings';
import type { DatabaseCreateDraft } from '@/entities/databases/model/DatabaseConnectionDraft';
import { DatabaseType } from '@/entities/databases/model/DatabaseType';

describe('changeDatabaseCreateDraftType', () => {
  it('resets the settings and the SSL choice and keeps the name and notifiers', () => {
    const postgresqlDraft: DatabaseCreateDraft = {
      scenario: 'create',
      workspaceId: 'workspace-id',
      name: 'Shop',
      notifierIds: ['notifier-id'],
      settings: {
        ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
        host: 'db.example.com',
        password: 'secret',
      },
      hasUserChosenSslMode: true,
    };

    const mysqlDraft = changeDatabaseCreateDraftType(postgresqlDraft, DatabaseType.MYSQL);

    expect(mysqlDraft.settings).toEqual(createDefaultDatabaseSettings(DatabaseType.MYSQL));
    expect(mysqlDraft.hasUserChosenSslMode).toBe(false);
    expect(mysqlDraft.name).toBe('Shop');
    expect(mysqlDraft.notifierIds).toEqual(['notifier-id']);
  });
});
