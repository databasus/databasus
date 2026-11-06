import { HealthStatus } from '@/entities/databases/model/HealthStatus';
import type { TranslationKey } from '@/shared/i18n';

export const HEALTH_STATUS_LABEL_KEYS: Record<HealthStatus, TranslationKey> = {
  [HealthStatus.AVAILABLE]: 'status.health.available',
  [HealthStatus.UNAVAILABLE]: 'status.health.unavailable',
};
