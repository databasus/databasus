import {
  type LucideIcon,
  ShieldAlert,
  ShieldCheck,
  ShieldEllipsis,
  ShieldOff,
  ShieldX,
} from 'lucide-react';

import { InstanceBackupHealth } from '@/entities/instance-backups';
import type { TranslationKey } from '@/shared/i18n';

export type InstanceBackupHealthColor = 'success' | 'accent' | 'warning' | 'danger';

export interface InstanceBackupHealthPresentation {
  color: InstanceBackupHealthColor;
  Icon: LucideIcon;
  titleKey: TranslationKey;
}

const HEALTH_PRESENTATIONS: Record<InstanceBackupHealth, InstanceBackupHealthPresentation> = {
  [InstanceBackupHealth.NOT_CONFIGURED]: {
    color: 'warning',
    Icon: ShieldOff,
    titleKey: 'instanceBackups.status.titles.notConfigured',
  },
  [InstanceBackupHealth.INITIALIZING]: {
    color: 'accent',
    Icon: ShieldEllipsis,
    titleKey: 'instanceBackups.status.titles.initializing',
  },
  [InstanceBackupHealth.HEALTHY]: {
    color: 'success',
    Icon: ShieldCheck,
    titleKey: 'instanceBackups.status.titles.healthy',
  },
  [InstanceBackupHealth.PARTIAL]: {
    color: 'warning',
    Icon: ShieldAlert,
    titleKey: 'instanceBackups.status.titles.partial',
  },
  [InstanceBackupHealth.FAILED]: {
    color: 'danger',
    Icon: ShieldX,
    titleKey: 'instanceBackups.status.titles.failed',
  },
  [InstanceBackupHealth.OVERDUE]: {
    color: 'warning',
    Icon: ShieldAlert,
    titleKey: 'instanceBackups.status.titles.overdue',
  },
  [InstanceBackupHealth.STOPPED]: {
    color: 'danger',
    Icon: ShieldOff,
    titleKey: 'instanceBackups.status.titles.stopped',
  },
  [InstanceBackupHealth.RESUMING]: {
    color: 'accent',
    Icon: ShieldEllipsis,
    titleKey: 'instanceBackups.status.titles.resuming',
  },
};

export const getInstanceBackupHealthPresentation = (
  health: InstanceBackupHealth,
): InstanceBackupHealthPresentation => HEALTH_PRESENTATIONS[health];
