import type { DatabaseSettings } from '@/entities/databases/model/DatabaseSettings';
import type { PostgresqlPhysicalSettings } from '@/entities/databases/model/ProviderDatabaseSettings';

// A physical backup is restored by other means, so a restore target never has that type.
export type RestoreTargetSettings = Exclude<DatabaseSettings, PostgresqlPhysicalSettings>;

export interface DatabaseCreateDraft {
  scenario: 'create';
  workspaceId: string;
  name: string;
  notifierIds: string[];
  settings: DatabaseSettings;
  hasUserChosenSslMode: boolean;
}

export interface DatabaseEditDraft {
  scenario: 'edit';
  workspaceId: string;
  databaseId: string;
  name: string;
  notifierIds: string[];
  savedSettings: DatabaseSettings;
  settings: DatabaseSettings;
  hasUserChosenSslMode: boolean;
}

// A restore target is usually another server, so this draft has no database identity: it cannot
// name a saved database whose stored secrets would fill what the settings leave out.
export interface RestoreTargetDraft {
  scenario: 'restore';
  settings: RestoreTargetSettings;
  hasUserChosenSslMode: boolean;
}

export type DatabaseConnectionDraft = DatabaseCreateDraft | DatabaseEditDraft | RestoreTargetDraft;
