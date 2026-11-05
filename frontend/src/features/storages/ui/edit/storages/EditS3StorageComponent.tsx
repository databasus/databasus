import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { Checkbox, Input, Radio, Select, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  S3StorageClass,
  S3_STORAGE_CLASS_LABEL_KEYS,
  type Storage,
} from '../../../../../entity/storages';
import { ApiError } from '../../../../../shared/api';
import { getWebsitePageUrl, useLocale } from '../../../../../shared/i18n';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setUnsaved: () => void;
  connectionError?: unknown;
}

export function EditS3StorageComponent({
  storage,
  setStorage,
  setUnsaved,
  connectionError,
}: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const hasAdvancedValues =
    !!storage?.s3Storage?.s3Prefix ||
    !!storage?.s3Storage?.s3UseVirtualHostedStyle ||
    !!storage?.s3Storage?.skipTLSVerify ||
    !!storage?.s3Storage?.s3StorageClass;
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  useEffect(() => {
    if (
      connectionError instanceof ApiError &&
      connectionError.message.includes('failed to verify certificate')
    ) {
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
          value={storage?.s3Storage?.s3Bucket || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Bucket: e.target.value.trim(),
              },
            });
            setUnsaved();
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
          value={storage?.s3Storage?.s3Region || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Region: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example region code
          placeholder="us-east-1"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.authType')}</div>
        <Radio.Group
          value={storage?.s3Storage?.authType || 'KEYS'}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                authType: e.target.value,
              },
            });
            setUnsaved();
          }}
          size="small"
        >
          <Radio.Button value="KEYS">{t('storages.s3AuthTypes.keys')}</Radio.Button>
          <Radio.Button value="IAM_ROLE">{t('storages.s3AuthTypes.iamRole')}</Radio.Button>
        </Radio.Group>
      </div>

      {(storage?.s3Storage?.authType || 'KEYS') === 'KEYS' ? (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.accessKey')}
            </div>
            <Input.Password
              value={storage?.s3Storage?.s3AccessKey || ''}
              onChange={(e) => {
                if (!storage?.s3Storage) return;

                setStorage({
                  ...storage,
                  s3Storage: {
                    ...storage.s3Storage,
                    s3AccessKey: e.target.value.trim(),
                  },
                });
                setUnsaved();
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
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.secretKey')}
            </div>
            <Input.Password
              value={storage?.s3Storage?.s3SecretKey || ''}
              onChange={(e) => {
                if (!storage?.s3Storage) return;

                setStorage({
                  ...storage,
                  s3Storage: {
                    ...storage.s3Storage,
                    s3SecretKey: e.target.value.trim(),
                  },
                });
                setUnsaved();
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
        </>
      ) : (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.roleArn')}</div>
            <div className="flex items-center">
              <Input
                value={storage?.s3Storage?.roleArn || ''}
                onChange={(e) => {
                  if (!storage?.s3Storage) return;

                  setStorage({
                    ...storage,
                    s3Storage: {
                      ...storage.s3Storage,
                      roleArn: e.target.value.trim(),
                    },
                  });
                  setUnsaved();
                }}
                size="small"
                className="w-full max-w-[250px]"
                // eslint-disable-next-line i18next/no-literal-string -- AWS example role ARN
                placeholder="arn:aws:iam::123456789012:role/RoleName"
              />
              <Tooltip className="cursor-pointer" title={t('storages.edit.s3.roleArnTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.externalId')}
            </div>
            <Input.Password
              value={storage?.s3Storage?.externalId || ''}
              onChange={(e) => {
                if (!storage?.s3Storage) return;

                setStorage({
                  ...storage,
                  s3Storage: {
                    ...storage.s3Storage,
                    externalId: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              placeholder={t('storages.edit.optionalPlaceholder', {
                // eslint-disable-next-line i18next/no-literal-string -- example external ID
                example: 'my-external-id',
              })}
              autoComplete="off"
            />
          </div>

          <div className="mb-2 text-xs text-gray-500">{t('storages.edit.s3.iamRoleNotice')}</div>
        </>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.endpoint')}</div>
        <div className="flex items-center">
          <Input
            value={storage?.s3Storage?.s3Endpoint || ''}
            onChange={(e) => {
              if (!storage?.s3Storage) return;

              setStorage({
                ...storage,
                s3Storage: {
                  ...storage.s3Storage,
                  s3Endpoint: e.target.value.trim(),
                },
              });
              setUnsaved();
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
                value={storage?.s3Storage?.s3Prefix || ''}
                onChange={(e) => {
                  if (!storage?.s3Storage) return;

                  setStorage({
                    ...storage,
                    s3Storage: {
                      ...storage.s3Storage,
                      s3Prefix: e.target.value.trim(),
                    },
                  });
                  setUnsaved();
                }}
                size="small"
                className="w-full max-w-[250px]"
                placeholder={t('storages.edit.optionalPlaceholder', { example: 'my-prefix/' })}
                // we do not allow to change the prefix after creation,
                // otherwise we will have to migrate all the data to the new prefix
                disabled={!!storage.id}
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
                checked={storage?.s3Storage?.s3UseVirtualHostedStyle || false}
                onChange={(e) => {
                  if (!storage?.s3Storage) return;

                  setStorage({
                    ...storage,
                    s3Storage: {
                      ...storage.s3Storage,
                      s3UseVirtualHostedStyle: e.target.checked,
                    },
                  });
                  setUnsaved();
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
                checked={storage?.s3Storage?.skipTLSVerify || false}
                onChange={(e) => {
                  if (!storage?.s3Storage) return;

                  setStorage({
                    ...storage,
                    s3Storage: {
                      ...storage.s3Storage,
                      skipTLSVerify: e.target.checked,
                    },
                  });
                  setUnsaved();
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
              <Select
                value={storage?.s3Storage?.s3StorageClass || S3StorageClass.DEFAULT}
                options={Object.values(S3StorageClass).map((storageClass) => ({
                  value: storageClass,
                  label: t(S3_STORAGE_CLASS_LABEL_KEYS[storageClass]),
                }))}
                onChange={(value) => {
                  if (!storage?.s3Storage) return;

                  setStorage({
                    ...storage,
                    s3Storage: {
                      ...storage.s3Storage,
                      s3StorageClass: value,
                    },
                  });
                  setUnsaved();
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
