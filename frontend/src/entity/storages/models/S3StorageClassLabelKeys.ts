import type { TranslationKey } from '../../../shared/i18n';
import { S3StorageClass } from './S3StorageClass';

export const S3_STORAGE_CLASS_LABEL_KEYS: Record<S3StorageClass, TranslationKey> = {
  [S3StorageClass.STANDARD]: 'storages.s3StorageClasses.standard',
  [S3StorageClass.STANDARD_IA]: 'storages.s3StorageClasses.standardIa',
  [S3StorageClass.ONEZONE_IA]: 'storages.s3StorageClasses.onezoneIa',
  [S3StorageClass.INTELLIGENT_TIERING]: 'storages.s3StorageClasses.intelligentTiering',
  [S3StorageClass.REDUCED_REDUNDANCY]: 'storages.s3StorageClasses.reducedRedundancy',
  [S3StorageClass.GLACIER_IR]: 'storages.s3StorageClasses.glacierIr',
};
