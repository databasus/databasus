import { DatabaseType, type RestoreTargetDraft } from '@/entities/databases';
import type { PostgresqlRestoreOptions, RestoreTarget } from '@/entities/restores';

// The target server of a restore is entered in the same form as the database's settings, so the
// target takes the variant of the engine the backup came from.
export const toRestoreTarget = (
  { settings }: RestoreTargetDraft,
  postgresqlRestoreOptions: PostgresqlRestoreOptions,
): RestoreTarget => {
  const password = settings.password?.trim() ?? '';

  switch (settings.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return {
        ...postgresqlRestoreOptions,
        type: settings.type,
        host: settings.host,
        port: settings.port,
        username: settings.username,
        password,
        database: settings.database ?? '',
        sslMode: settings.sslMode,
        sslClientCert: settings.sslClientCert,
        sslClientKey: settings.sslClientKey,
        sslRootCert: settings.sslRootCert,
        sshTunnel: settings.sshTunnel,
        cpuCount: settings.cpuCount,
      };
    case DatabaseType.MYSQL:
      return {
        type: settings.type,
        host: settings.host,
        port: settings.port,
        username: settings.username,
        password,
        database: settings.database ?? '',
        isHttps: settings.isHttps,
        sshTunnel: settings.sshTunnel,
      };
    case DatabaseType.MARIADB:
      return {
        type: settings.type,
        host: settings.host,
        port: settings.port,
        username: settings.username,
        password,
        database: settings.database ?? '',
        isHttps: settings.isHttps,
        isSkipGaleraDisable: settings.isSkipGaleraDisable,
        sshTunnel: settings.sshTunnel,
      };
    case DatabaseType.MONGODB:
      return {
        type: settings.type,
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
        sshTunnel: settings.sshTunnel,
      };
  }
};
