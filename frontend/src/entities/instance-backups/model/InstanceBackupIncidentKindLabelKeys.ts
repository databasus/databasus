import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

type IncidentKind = ApiComponents['schemas']['InstanceBackupIncidentKind'];

export const INSTANCE_BACKUP_INCIDENT_KIND_LABEL_KEYS: Record<IncidentKind, TranslationKey> = {
  FAILED: 'instanceBackups.incidents.failed',
  PARTIAL: 'instanceBackups.incidents.partial',
  OVERDUE: 'instanceBackups.incidents.overdue',
  STOPPED: 'instanceBackups.incidents.stopped',
};
