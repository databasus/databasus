import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { AutoComplete, Checkbox, Input, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  S3_STORAGE_CLASS_LABEL_KEYS,
  type S3StorageSettings,
} from '../../../../../entities/storages';
import { getWebsitePageUrl, useLocale } from '../../../../../shared/i18n';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: S3StorageSettings;
  isSaved: boolean;
  onChange: (settings: S3StorageSettings) => void;
  // The error of the latest failed connection test.
  connectionError?: string;
}

export function EditS3StorageComponent({ settings, isSaved, onChange, connectionError }: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const hasAdvancedValues =
    !!settings.s3Prefix ||
    !!settings.isVirtualHostedStyle ||
    !!settings.isSkipTlsVerify ||
    !!settings.s3StorageClass;
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  useEffect(() => {
    if (connectionError?.includes('failed to verify certificate')) {
      setShowAdvanced(true);
    }
  }, [connectionError]);

  return (
    <>
      <div className="mb-2 flex items-center">
        <div className="hidden min-w-[110px] sm:block" />

        <div className="text-xs text-blue-600">
          <a
            href={getWebsitePageUrl('storagesCloudflareR2', locale)}
            target="_blank"
            rel="noreferrer"
          >
            {t('storages.edit.s3.cloudflareR2Guide')}
          </a>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.s3Bucket')}</div>
        <Input
          value={settings.s3Bucket || ''}
          onChange={(e) => {
            onChange({ ...settings, s3Bucket: e.target.value.trim() });
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example bucket name
          placeholder="my-bucket-name"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.region')}</div>
        <Input
          value={settings.s3Region || ''}
          onChange={(e) => {
            onChange({ ...settings, s3Region: toOptionalValue(e.target.value.trim()) });
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example region code
          placeholder="us-east-1"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.accessKey')}</div>
        <Input.Password
          value={settings.s3AccessKey || ''}
          onChange={(e) => {
            onChange({ ...settings, s3AccessKey: toOptionalValue(e.target.value.trim()) });
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- AWS example access key
          placeholder="AKIAIOSFODNN7EXAMPLE"
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.secretKey')}</div>
        <Input.Password
          value={settings.s3SecretKey || ''}
          onChange={(e) => {
            onChange({ ...settings, s3SecretKey: toOptionalValue(e.target.value.trim()) });
          }}
          size="small"
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- AWS example secret key
          placeholder="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.endpoint')}</div>
        <div className="flex items-center">
          <Input
            value={settings.s3Endpoint || ''}
            onChange={(e) => {
              onChange({ ...settings, s3Endpoint: toOptionalValue(e.target.value.trim()) });
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.optionalPlaceholder', {
              example: 'https://s3.example.com',
            })}
          />

          <Tooltip className="cursor-pointer" title={t('storages.edit.s3.endpointTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mt-4 mb-3 flex items-center">
        <div
          className="flex cursor-pointer items-center text-sm text-blue-600 hover:text-blue-800"
          onClick={() => setShowAdvanced(!showAdvanced)}
        >
          <span className="mr-2">{t('storages.edit.advancedSettings')}</span>

          {showAdvanced ? (
            <UpOutlined style={{ fontSize: '12px' }} />
          ) : (
            <DownOutlined style={{ fontSize: '12px' }} />
          )}
        </div>
      </div>

      {showAdvanced && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.folderPrefix')}
            </div>
            <div className="flex items-center">
              <Input
                value={settings.s3Prefix || ''}
                onChange={(e) => {
                  onChange({ ...settings, s3Prefix: toOptionalValue(e.target.value.trim()) });
                }}
                size="small"
                className="w-full max-w-[250px]"
                placeholder={t('storages.edit.optionalPlaceholder', { example: 'my-prefix/' })}
                // we do not allow to change the prefix after creation,
                // otherwise we will have to migrate all the data to the new prefix
                disabled={isSaved}
              />

              <Tooltip className="cursor-pointer" title={t('storages.edit.s3.prefixTooltip')}>
                <InfoCircleOutlined className="ml-4" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.virtualHost')}
            </div>
            <div className="flex items-center">
              <Checkbox
                checked={settings.isVirtualHostedStyle || false}
                onChange={(e) => {
                  onChange({ ...settings, isVirtualHostedStyle: e.target.checked });
                }}
              >
                {t('storages.edit.s3.useVirtualHostedStyle')}
              </Checkbox>

              <Tooltip className="cursor-pointer" title={t('storages.edit.s3.virtualHostTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.skipTlsVerify')}
            </div>
            <div className="flex items-center">
              <Checkbox
                checked={settings.isSkipTlsVerify || false}
                onChange={(e) => {
                  onChange({ ...settings, isSkipTlsVerify: e.target.checked });
                }}
              >
                {t('storages.edit.s3.skipTls')}
              </Checkbox>

              <Tooltip
                className="cursor-pointer"
                title={t('storages.edit.s3.skipTlsVerifyTooltip')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.storageClass')}
            </div>
            <div className="flex items-center">
              <AutoComplete
                aria-label={t('storages.fields.storageClass')}
                value={settings.s3StorageClass ?? ''}
                placeholder={t('storages.s3StorageClasses.default')}
                options={[
                  {
                    value: '',
                    label: t('storages.s3StorageClasses.default'),
                  },
                  ...Array.from(S3_STORAGE_CLASS_LABEL_KEYS, ([storageClass, labelKey]) => ({
                    value: storageClass,
                    label: t(labelKey),
                  })),
                ]}
                onChange={(value: string) => {
                  onChange({
                    ...settings,
                    s3StorageClass: value === '' ? undefined : value,
                  });
                }}
                size="small"
                className="w-[250px] max-w-[250px]"
              />

              <Tooltip className="cursor-pointer" title={t('storages.edit.s3.storageClassTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        </>
      )}

      <div className="mb-5" />
    </>
  );
}
