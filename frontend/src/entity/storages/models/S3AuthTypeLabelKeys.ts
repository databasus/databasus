import type { TranslationKey } from '../../../shared/i18n';
import type { S3AuthType } from './S3Storage';

export const S3_AUTH_TYPE_LABEL_KEYS: Record<S3AuthType, TranslationKey> = {
  KEYS: 'storages.s3AuthTypes.keys',
  IAM_ROLE: 'storages.s3AuthTypes.iamRole',
};
