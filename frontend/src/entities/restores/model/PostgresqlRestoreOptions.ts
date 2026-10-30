import type { ApiComponents } from '../../../shared/api';

// The restore options of a PostgreSQL target, which the restore form asks for beside the
// connection settings.
export type PostgresqlRestoreOptions = Pick<
  ApiComponents['schemas']['PostgresqlRestoreTarget'],
  'isExcludeExtensions' | 'isRestoreOwnership' | 'isRestorePrivileges'
>;
