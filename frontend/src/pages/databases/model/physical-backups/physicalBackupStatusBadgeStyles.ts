/* eslint-disable i18next/no-literal-string -- Tailwind class names */
import { PhysicalBackupStatus } from './PhysicalBackupStatus';

interface PhysicalBackupStatusBadgeStyle {
  pillClasses: string;
  dotClasses: string;
}

export const PHYSICAL_BACKUP_STATUS_BADGE_STYLES: Record<
  PhysicalBackupStatus,
  PhysicalBackupStatusBadgeStyle
> = {
  [PhysicalBackupStatus.COMPLETED]: {
    pillClasses: 'bg-success/10 text-success ring-success/30',
    dotClasses: 'bg-success-fill',
  },
  [PhysicalBackupStatus.IN_PROGRESS]: {
    pillClasses: 'bg-accent/10 text-accent ring-accent/30',
    dotClasses: 'bg-accent',
  },
  [PhysicalBackupStatus.ERROR]: {
    pillClasses: 'bg-danger/10 text-danger ring-danger/30',
    dotClasses: 'bg-danger',
  },
  [PhysicalBackupStatus.CHAIN_BROKEN]: {
    pillClasses: 'bg-warning/10 text-warning ring-warning/30',
    dotClasses: 'bg-warning',
  },
  [PhysicalBackupStatus.CANCELED]: {
    pillClasses: 'bg-muted/10 text-muted ring-muted/30',
    dotClasses: 'bg-muted',
  },
};
