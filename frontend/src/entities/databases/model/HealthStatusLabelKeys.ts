import type { TranslationKey } from '../../../shared/i18n';
import { HealthStatus } from './HealthStatus';

export const HEALTH_STATUS_LABEL_KEYS: Record<HealthStatus, TranslationKey> = {
  [HealthStatus.AVAILABLE]: 'status.health.available',
  [HealthStatus.UNAVAILABLE]: 'status.health.unavailable',
};
