import type { DatabaseSettings } from './DatabaseSettings';
import type { DatabaseType } from './DatabaseType';
import { PhysicalDatabaseBackupType } from './postgresql/physical/PhysicalDatabaseBackupType';
import { PostgresSslMode } from './postgresql/PostgresSslMode';
import { createEmptySshTunnelSettings } from './sshtunnel/createEmptySshTunnelSettings';

const DEFAULT_POSTGRESQL_PORT = 5432;
const DEFAULT_MYSQL_PORT = 3306;
const DEFAULT_MONGODB_PORT = 27017;

type SettingsOfType<Type extends DatabaseType> = Extract<DatabaseSettings, { type: Type }>;

const createConnection = () => ({
  host: '',
  username: '',
  password: '',
  sshTunnel: createEmptySshTunnelSettings(),
});

const DEFAULT_SETTINGS_BUILDERS: { [Type in DatabaseType]: () => SettingsOfType<Type> } = {
  POSTGRES_LOGICAL: () => ({
    ...createConnection(),
    type: 'POSTGRES_LOGICAL',
    port: DEFAULT_POSTGRESQL_PORT,
    database: '',
    sslMode: PostgresSslMode.Disable,
    includeSchemas: [],
    excludeTables: [],
    excludeTableData: [],
    cpuCount: 1,
    isSkipUserMappings: false,
  }),
  POSTGRES_PHYSICAL: () => ({
    ...createConnection(),
    type: 'POSTGRES_PHYSICAL',
    port: DEFAULT_POSTGRESQL_PORT,
    sslMode: PostgresSslMode.Disable,
    backupType: PhysicalDatabaseBackupType.FULL,
  }),
  MYSQL: () => ({
    ...createConnection(),
    type: 'MYSQL',
    port: DEFAULT_MYSQL_PORT,
    database: '',
    isHttps: false,
    excludeTables: [],
  }),
  MARIADB: () => ({
    ...createConnection(),
    type: 'MARIADB',
    port: DEFAULT_MYSQL_PORT,
    database: '',
    isHttps: false,
    excludeTables: [],
    isExcludeEvents: false,
    isSkipGaleraDisable: false,
  }),
  MONGODB: () => ({
    ...createConnection(),
    type: 'MONGODB',
    port: DEFAULT_MONGODB_PORT,
    database: '',
    authDatabase: '',
    isHttps: false,
    isSrv: false,
    isDirectConnection: false,
    cpuCount: 1,
    excludeCollections: [],
  }),
};

// The settings a new database of the type starts the form with.
export const createDefaultDatabaseSettings = <Type extends DatabaseType>(
  type: Type,
): SettingsOfType<Type> => DEFAULT_SETTINGS_BUILDERS[type]();
