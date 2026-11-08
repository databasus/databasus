import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type LogicalRetentionPolicy = ApiComponents['schemas']['LogicalRetentionPolicy'];

export const LogicalRetentionPolicy = {
  TIME_PERIOD: 'TIME_PERIOD',
  COUNT: 'COUNT',
  GFS: 'GFS',
} as const satisfies Record<string, LogicalRetentionPolicy>;

export const LOGICAL_RETENTION_POLICY_LABEL_KEYS: Record<LogicalRetentionPolicy, TranslationKey> = {
  [LogicalRetentionPolicy.GFS]: 'backups.logical.retentionPolicyTypes.gfs',
  [LogicalRetentionPolicy.TIME_PERIOD]: 'backups.logical.retentionPolicyTypes.timePeriod',
  [LogicalRetentionPolicy.COUNT]: 'backups.logical.retentionPolicyTypes.count',
};
