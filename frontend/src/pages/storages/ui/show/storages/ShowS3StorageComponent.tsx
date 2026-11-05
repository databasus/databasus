import { useTranslation } from 'react-i18next';

import {
  S3_STORAGE_CLASS_LABEL_KEYS,
  type S3StorageSettings,
} from '../../../../../entities/storages';

interface Props {
  settings: S3StorageSettings;
}

export function ShowS3StorageComponent({ settings }: Props) {
  const { t } = useTranslation();
  const storageClassLabelKey = settings.s3StorageClass
    ? S3_STORAGE_CLASS_LABEL_KEYS.get(settings.s3StorageClass)
    : undefined;

  return (
    <>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.s3Bucket')}</div>
        {settings.s3Bucket}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.region')}</div>
        {settings.s3Region || '-'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.accessKey')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.secretKey')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.endpoint')}</div>
        {settings.s3Endpoint || '-'}
      </div>

      {settings.s3Prefix && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.prefix')}</div>
          {settings.s3Prefix}
        </div>
      )}

      {settings.isVirtualHostedStyle && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.virtualHost')}</div>
          {t('common.states.enabled')}
        </div>
      )}

      {settings.isSkipTlsVerify && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.skipTls')}</div>
          {t('common.states.enabled')}
        </div>
      )}

      {settings.s3StorageClass && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.storageClass')}</div>
          {storageClassLabelKey ? t(storageClassLabelKey) : settings.s3StorageClass}
        </div>
      )}
    </>
  );
}
