import { useTranslation } from 'react-i18next';

import {
  S3StorageClass,
  S3_STORAGE_CLASS_LABEL_KEYS,
  type Storage,
} from '../../../../../entity/storages';
import type { TranslationKey } from '../../../../../shared/i18n';

interface Props {
  storage: Storage;
}

export function ShowS3StorageComponent({ storage }: Props) {
  const { t } = useTranslation();

  const storageClassLabelKey: TranslationKey | undefined =
    S3_STORAGE_CLASS_LABEL_KEYS[storage?.s3Storage?.s3StorageClass as S3StorageClass];

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.s3Bucket')}</div>
        {storage?.s3Storage?.s3Bucket}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.region')}</div>
        {storage?.s3Storage?.s3Region || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.authType')}</div>
        {storage?.s3Storage?.authType === 'IAM_ROLE'
          ? t('storages.s3AuthTypes.iamRole')
          : t('storages.s3AuthTypes.keys')}
      </div>

      {storage?.s3Storage?.authType === 'IAM_ROLE' ? (
        storage?.s3Storage?.roleArn ? (
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('storages.fields.roleArn')}</div>
            {storage.s3Storage.roleArn}
          </div>
        ) : null
      ) : (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('storages.fields.accessKey')}</div>
            {'*************'}
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('storages.fields.secretKey')}</div>
            {'*************'}
          </div>
        </>
      )}

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.endpoint')}</div>
        {storage?.s3Storage?.s3Endpoint || '-'}
      </div>

      {storage?.s3Storage?.s3Prefix && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('storages.fields.prefix')}</div>
          {storage.s3Storage.s3Prefix}
        </div>
      )}

      {storage?.s3Storage?.s3UseVirtualHostedStyle && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('storages.fields.virtualHost')}</div>
          {t('common.states.enabled')}
        </div>
      )}

      {storage?.s3Storage?.skipTLSVerify && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('storages.fields.skipTls')}</div>
          {t('common.states.enabled')}
        </div>
      )}

      {storage?.s3Storage?.s3StorageClass && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('storages.fields.storageClass')}</div>
          {storageClassLabelKey ? t(storageClassLabelKey) : storage.s3Storage.s3StorageClass}
        </div>
      )}
    </>
  );
}
