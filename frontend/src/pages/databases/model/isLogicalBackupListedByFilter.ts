import dayjs from 'dayjs';

import type { LogicalBackup } from '@/pages/databases/model/LogicalBackup';
import type { LogicalBackupListFilter } from '@/pages/databases/model/LogicalBackupListFilter';

// Whether the server would list the backup under the filter, so a backup a request answered with
// is shown before the next refresh only where the list would show it.
export const isLogicalBackupListedByFilter = (
  backup: LogicalBackup,
  filter: LogicalBackupListFilter,
): boolean => {
  const isStatusListed =
    !filter.statuses || filter.statuses.length === 0 || filter.statuses.includes(backup.status);
  const isTimeListed =
    filter.createdBefore === undefined || dayjs(backup.createdAt).isBefore(filter.createdBefore);

  return isStatusListed && isTimeListed;
};
