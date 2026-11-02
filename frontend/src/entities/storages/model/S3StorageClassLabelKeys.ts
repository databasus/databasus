import type { TranslationKey } from '../../../shared/i18n';

export const S3_STORAGE_CLASS_LABEL_KEYS: ReadonlyMap<string, TranslationKey> = new Map([
  ['STANDARD', 'storages.s3StorageClasses.standard'],
  ['STANDARD_IA', 'storages.s3StorageClasses.standardIa'],
  ['ONEZONE_IA', 'storages.s3StorageClasses.onezoneIa'],
  ['INTELLIGENT_TIERING', 'storages.s3StorageClasses.intelligentTiering'],
  ['REDUCED_REDUNDANCY', 'storages.s3StorageClasses.reducedRedundancy'],
  ['GLACIER_IR', 'storages.s3StorageClasses.glacierIr'],
]);
