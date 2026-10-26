import { type DatabaseSettings, DatabaseType } from '../../../entity/databases';
import type {
  PostgresqlRestoreOptions,
  RestoreBackupRequest,
  RestoreTargetSshTunnel,
} from '../../../entity/restores';

const NO_RESTORE_OPTIONS: PostgresqlRestoreOptions = {
  isExcludeExtensions: false,
  isRestoreOwnership: false,
  isRestorePrivileges: false,
};

const toRestoreTargetSshTunnel = (
  sshTunnel: DatabaseSettings['sshTunnel'],
): RestoreTargetSshTunnel => ({ ...sshTunnel });

// The target server of a restore is entered in the same form as the database's settings; the
// request takes the connection fields of the engine the backup came from. A physical backup is
// restored by other means, so it has no target here.
export const toRestoreBackupRequest = (
  settings: DatabaseSettings,
  postgresqlRestoreOptions: PostgresqlRestoreOptions = NO_RESTORE_OPTIONS,
): RestoreBackupRequest => {
  const sshTunnel = toRestoreTargetSshTunnel(settings.sshTunnel);
  const password = settings.password?.trim();

  switch (settings.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return {
        postgresqlDatabase: {
          ...postgresqlRestoreOptions,
          version: settings.version,
          host: settings.host,
          port: settings.port,
          username: settings.username,
          password,
          database: settings.database,
          sslMode: settings.sslMode,
          sslClientCert: settings.sslClientCert,
          sslClientKey: settings.sslClientKey,
          sslRootCert: settings.sslRootCert,
          sshTunnel,
          cpuCount: settings.cpuCount,
        },
      };
    case DatabaseType.MYSQL:
      return {
        mysqlDatabase: {
          version: settings.version,
          host: settings.host,
          port: settings.port,
          username: settings.username,
          password,
          database: settings.database,
          isHttps: settings.isHttps,
          sshTunnel,
        },
      };
    case DatabaseType.MARIADB:
      return {
        mariadbDatabase: {
          version: settings.version,
          host: settings.host,
          port: settings.port,
          username: settings.username,
          password,
          database: settings.database,
          isHttps: settings.isHttps,
          isSkipGaleraDisable: settings.isSkipGaleraDisable,
          sshTunnel,
        },
      };
    case DatabaseType.MONGODB:
      return {
        mongodbDatabase: {
          version: settings.version,
          host: settings.host,
          port: settings.port,
          username: settings.username,
          password,
          database: settings.database,
          authDatabase: settings.authDatabase,
          isHttps: settings.isHttps,
          isSrv: settings.isSrv,
          isDirectConnection: settings.isDirectConnection,
          cpuCount: settings.cpuCount,
          sshTunnel,
        },
      };
    case DatabaseType.POSTGRES_PHYSICAL:
      return {};
  }
};
