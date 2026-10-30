import { createDefaultDatabaseSettings } from './createDefaultDatabaseSettings';
import type { Database } from './Database';
import type { DatabaseType } from './DatabaseType';

// Going back and forth in the creation wizard keeps the settings already entered for the type.
export const changeDatabaseType = (database: Database, type: DatabaseType): Database =>
  database.settings.type === type
    ? database
    : { ...database, settings: createDefaultDatabaseSettings(type) };
