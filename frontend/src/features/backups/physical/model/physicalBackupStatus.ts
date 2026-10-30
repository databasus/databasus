/* eslint-disable i18next/no-literal-string -- Tailwind class names */
import { PhysicalBackupStatus } from '../../../../entities/backups/physical';

interface PhysicalBackupStatusBadgeStyle {
  pillClasses: string;
  dotClasses: string;
}

export const PHYSICAL_BACKUP_STATUS_BADGE_STYLES: Record<
  PhysicalBackupStatus,
  PhysicalBackupStatusBadgeStyle
> = {
  [PhysicalBackupStatus.COMPLETED]: {
    pillClasses:
      'bg-green-500/10 text-green-700 ring-green-500/30 dark:bg-green-500/15 dark:text-green-300 dark:ring-green-400/30',
    dotClasses: 'bg-green-500 dark:bg-green-400',
  },
  [PhysicalBackupStatus.IN_PROGRESS]: {
    pillClasses:
      'bg-blue-500/10 text-blue-700 ring-blue-500/30 dark:bg-blue-500/15 dark:text-blue-300 dark:ring-blue-400/30',
    dotClasses: 'bg-blue-500 dark:bg-blue-400',
  },
  [PhysicalBackupStatus.ERROR]: {
    pillClasses:
      'bg-rose-500/10 text-rose-700 ring-rose-500/30 dark:bg-rose-500/15 dark:text-rose-300 dark:ring-rose-400/30',
    dotClasses: 'bg-rose-500 dark:bg-rose-400',
  },
  [PhysicalBackupStatus.CHAIN_BROKEN]: {
    pillClasses:
      'bg-amber-500/10 text-amber-700 ring-amber-500/30 dark:bg-amber-500/15 dark:text-amber-300 dark:ring-amber-400/30',
    dotClasses: 'bg-amber-500 dark:bg-amber-400',
  },
  [PhysicalBackupStatus.CANCELED]: {
    pillClasses:
      'bg-gray-500/10 text-gray-700 ring-gray-500/30 dark:bg-gray-500/15 dark:text-gray-300 dark:ring-gray-400/30',
    dotClasses: 'bg-gray-500 dark:bg-gray-400',
  },
};
