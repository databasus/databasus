import { LogicalRetentionPolicy } from '@/pages/databases/model/LogicalRetentionPolicy';
import type { TranslationKey } from '@/shared/i18n';

export const LOGICAL_RETENTION_POLICY_LABEL_KEYS: Record<LogicalRetentionPolicy, TranslationKey> = {
  [LogicalRetentionPolicy.GFS]: 'backups.logical.retentionPolicyTypes.gfs',
  [LogicalRetentionPolicy.TIME_PERIOD]: 'backups.logical.retentionPolicyTypes.timePeriod',
  [LogicalRetentionPolicy.COUNT]: 'backups.logical.retentionPolicyTypes.count',
};
