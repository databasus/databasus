import type { ApiComponents } from '../../../shared/api';

type Schemas = ApiComponents['schemas'];

// The restore operation is not in the API contract yet, so these shapes follow the JSON field
// names the old restore handler reads.
export interface RestoreTargetSshTunnel {
  isEnabled: boolean;
  host?: string;
  port?: number;
  username?: string;
  authType?: Schemas['SshAuthType'];
  password?: string;
  privateKey?: string;
  privateKeyPassphrase?: string;
}

export interface PostgresqlRestoreOptions {
  isExcludeExtensions: boolean;
  isRestoreOwnership: boolean;
  isRestorePrivileges: boolean;
}

export interface PostgresqlRestoreTarget extends PostgresqlRestoreOptions {
  version?: Schemas['PostgresqlVersion'];
  host: string;
  port: number;
  username: string;
  password?: string;
  database?: string;
  sslMode: Schemas['PostgresqlSslMode'];
  sslClientCert?: string;
  sslClientKey?: string;
  sslRootCert?: string;
  sshTunnel: RestoreTargetSshTunnel;
  cpuCount: number;
}

export interface MysqlRestoreTarget {
  version?: Schemas['MysqlVersion'];
  host: string;
  port: number;
  username: string;
  password?: string;
  database?: string;
  isHttps: boolean;
  sshTunnel: RestoreTargetSshTunnel;
}

export interface MariadbRestoreTarget {
  version?: Schemas['MariadbVersion'];
  host: string;
  port: number;
  username: string;
  password?: string;
  database?: string;
  isHttps: boolean;
  isSkipGaleraDisable: boolean;
  sshTunnel: RestoreTargetSshTunnel;
}

export interface MongodbRestoreTarget {
  version?: Schemas['MongodbVersion'];
  host: string;
  port?: number;
  username: string;
  password?: string;
  database: string;
  authDatabase: string;
  isHttps: boolean;
  isSrv: boolean;
  isDirectConnection: boolean;
  cpuCount: number;
  sshTunnel: RestoreTargetSshTunnel;
}

// One target is set, the one of the backed-up database's engine.
export interface RestoreBackupRequest {
  postgresqlDatabase?: PostgresqlRestoreTarget;
  mysqlDatabase?: MysqlRestoreTarget;
  mariadbDatabase?: MariadbRestoreTarget;
  mongodbDatabase?: MongodbRestoreTarget;
}
