import { InstanceBackupHealth } from '@/entities/instance-backups/model/InstanceBackupHealth';
import type { TranslationKey } from '@/shared/i18n';

export const INSTANCE_BACKUP_HEALTH_LABEL_KEYS: Record<InstanceBackupHealth, TranslationKey> = {
  [InstanceBackupHealth.NOT_CONFIGURED]: 'instanceBackups.health.notConfigured',
  [InstanceBackupHealth.INITIALIZING]: 'instanceBackups.health.initializing',
  [InstanceBackupHealth.HEALTHY]: 'instanceBackups.health.healthy',
  [InstanceBackupHealth.PARTIAL]: 'instanceBackups.health.partial',
  [InstanceBackupHealth.FAILED]: 'instanceBackups.health.failed',
  [InstanceBackupHealth.OVERDUE]: 'instanceBackups.health.overdue',
  [InstanceBackupHealth.STOPPED]: 'instanceBackups.health.stopped',
  [InstanceBackupHealth.RESUMING]: 'instanceBackups.health.resuming',
};
