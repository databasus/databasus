import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Radio, Tooltip } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  AZURE_BLOB_AUTH_METHOD_LABEL_KEYS,
  type AzureBlobStorageSettings,
} from '@/entities/storages';
import { toOptionalValue } from '@/shared/lib';
import { AddressChangedSecretNoticeComponent, AdvancedSettingsToggleComponent } from '@/shared/ui';

interface Props {
  settings: AzureBlobStorageSettings;
  isSaved: boolean;
  isAddressChanged: boolean;
  onChange: (settings: AzureBlobStorageSettings) => void;
}

export function EditAzureBlobStorageComponent({
  settings,
  isSaved,
  isAddressChanged,
  onChange,
}: Props) {
  const { t } = useTranslation();
  const authMethodLabelId = useId();
  const hasAdvancedValues = !!settings.prefix || !!settings.endpoint;
  const [isShowAdvanced, setIsShowAdvanced] = useState(hasAdvancedValues);
  const advancedRegionId = useId();
  const accountKeyNoticeId = useId();
  const isAccountKeyRequiredAgain = isAddressChanged && !settings.accountKey;

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div id={authMethodLabelId} className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
          {t('storages.fields.authMethod')}
        </div>
        <div role="radiogroup" aria-labelledby={authMethodLabelId}>
          <Radio.Group
            value={settings.authMethod || 'ACCOUNT_KEY'}
            onChange={(e) => {
              onChange({ ...settings, authMethod: e.target.value });
            }}
            size="small"
          >
            <Radio value="ACCOUNT_KEY">{t(AZURE_BLOB_AUTH_METHOD_LABEL_KEYS.ACCOUNT_KEY)}</Radio>
            <Radio value="CONNECTION_STRING">
              {t(AZURE_BLOB_AUTH_METHOD_LABEL_KEYS.CONNECTION_STRING)}
            </Radio>
          </Radio.Group>
        </div>
      </div>

      {settings.authMethod === 'CONNECTION_STRING' && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
            {t('storages.fields.connection')}
          </div>
          <div className="flex items-center">
            <Input.Password
              aria-label={t('storages.fields.connection')}
              value={settings.connectionString || ''}
              onChange={(e) => {
                onChange({ ...settings, connectionString: toOptionalValue(e.target.value.trim()) });
              }}
              size="small"
              className="w-full max-w-[250px]"
              // eslint-disable-next-line i18next/no-literal-string -- connection string format example
              placeholder="DefaultEndpointsProtocol=https;AccountName=..."
              autoComplete="off"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
            />

            <Tooltip
              className="cursor-pointer"
              title={t('storages.edit.azureBlob.connectionStringTooltip')}
            >
              <InfoCircleOutlined className="ml-2" />
            </Tooltip>
          </div>
        </div>
      )}

      {settings.authMethod === 'ACCOUNT_KEY' && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.accountName')}
            </div>
            <Input
              aria-label={t('storages.fields.accountName')}
              value={settings.accountName || ''}
              onChange={(e) => {
                onChange({ ...settings, accountName: toOptionalValue(e.target.value.trim()) });
              }}
              size="small"
              className="w-full max-w-[250px]"
              // eslint-disable-next-line i18next/no-literal-string -- example account name
              placeholder="mystorageaccount"
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.accountKey')}
            </div>
            <div className="w-full max-w-[250px]">
              <Input.Password
                aria-label={t('storages.fields.accountKey')}
                aria-describedby={isAccountKeyRequiredAgain ? accountKeyNoticeId : undefined}
                value={settings.accountKey || ''}
                onChange={(e) => {
                  onChange({ ...settings, accountKey: toOptionalValue(e.target.value.trim()) });
                }}
                size="small"
                className="w-full"
                // eslint-disable-next-line i18next/no-literal-string -- example key value
                placeholder="your-account-key"
                autoComplete="off"
                data-1p-ignore
                data-lpignore="true"
                data-form-type="other"
              />
              {isAccountKeyRequiredAgain && (
                <AddressChangedSecretNoticeComponent id={accountKeyNoticeId} />
              )}
            </div>
          </div>
        </>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
          {t('storages.fields.containerName')}
        </div>
        <Input
          aria-label={t('storages.fields.containerName')}
          value={settings.containerName || ''}
          onChange={(e) => {
            onChange({ ...settings, containerName: e.target.value.trim() });
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example container name
          placeholder="my-container"
        />
      </div>

      <AdvancedSettingsToggleComponent
        label={t('storages.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
        regionId={advancedRegionId}
      />

      {isShowAdvanced && (
        <div id={advancedRegionId}>
          {settings.authMethod === 'ACCOUNT_KEY' && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
                {t('storages.fields.endpoint')}
              </div>
              <div className="flex items-center">
                <Input
                  aria-label={t('storages.fields.endpoint')}
                  value={settings.endpoint || ''}
                  onChange={(e) => {
                    onChange({ ...settings, endpoint: toOptionalValue(e.target.value.trim()) });
                  }}
                  size="small"
                  className="w-full max-w-[250px]"
                  placeholder={t('storages.edit.optionalPlaceholder', {
                    example: 'https://myaccount.blob.core.windows.net',
                  })}
                />

                <Tooltip
                  className="cursor-pointer"
                  title={t('storages.edit.azureBlob.endpointTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" />
                </Tooltip>
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.blobPrefix')}
            </div>
            <div className="flex items-center">
              <Input
                aria-label={t('storages.fields.blobPrefix')}
                value={settings.prefix || ''}
                onChange={(e) => {
                  onChange({ ...settings, prefix: toOptionalValue(e.target.value.trim()) });
                }}
                size="small"
                className="w-full max-w-[250px]"
                placeholder={t('storages.edit.optionalPlaceholder', { example: 'my-prefix/' })}
                // Every stored blob is named under the prefix, so it is fixed once the storage is saved
                disabled={isSaved}
              />

              <Tooltip
                className="cursor-pointer"
                title={t('storages.edit.azureBlob.prefixTooltip')}
              >
                <InfoCircleOutlined className="ml-2" />
              </Tooltip>
            </div>
          </div>
        </div>
      )}

      <div className="mb-5" />
    </>
  );
}
