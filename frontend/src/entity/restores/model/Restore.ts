import { RestoreStatus } from './RestoreStatus';
import type { PostgresqlRestoreTarget } from './RestoreTarget';

export interface Restore {
  id: string;
  status: RestoreStatus;

  postgresqlDatabase?: PostgresqlRestoreTarget;

  failMessage?: string;

  restoreDurationMs: number;
  createdAt: string;
}
