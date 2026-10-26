import type { DatabaseNotifier } from './DatabaseNotifier';
import type { DatabaseType } from './DatabaseType';
import type { HealthStatus } from './HealthStatus';
import type { MariadbDatabase } from './mariadb/MariadbDatabase';
import type { MongodbDatabase } from './mongodb/MongodbDatabase';
import type { MysqlDatabase } from './mysql/MysqlDatabase';
import type { PostgresqlPhysicalDatabase } from './postgresql/physical/PostgresqlPhysicalDatabase';
import type { PostgresqlLogicalDatabase } from './postgresql/PostgresqlLogicalDatabase';

export interface Database {
  id: string;
  name: string;
  workspaceId: string;
  type: DatabaseType;

  postgresqlLogical?: PostgresqlLogicalDatabase;
  postgresqlPhysical?: PostgresqlPhysicalDatabase;
  mysql?: MysqlDatabase;
  mariadb?: MariadbDatabase;
  mongodb?: MongodbDatabase;

  notifiers: DatabaseNotifier[];

  lastBackupTime?: Date;
  lastBackupErrorMessage?: string;

  healthStatus?: HealthStatus;
}
