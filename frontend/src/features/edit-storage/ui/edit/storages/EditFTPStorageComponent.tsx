import { InfoCircleOutlined } from '@ant-design/icons';
import { Checkbox, Input, InputNumber, Tooltip } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { AddressChangedSecretNoticeComponent, AdvancedSettingsToggleComponent } from '@/shared/ui';

import type { FtpStorageSettings } from '../../../../../entities/storages';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: FtpStorageSettings;
  isAddressChanged: boolean;
  onChange: (settings: FtpStorageSettings) => void;
}

export function EditFTPStorageComponent({ settings, isAddressChanged, onChange }: Props) {
  const { t } = useTranslation();
  const hasAdvancedValues = !!settings.isSkipTlsVerify;
  const [isShowAdvanced, setIsShowAdvanced] = useState(hasAdvancedValues);
  const advancedRegionId = useId();
  const passwordNoticeId = useId();
  const isPasswordRequiredAgain = isAddressChanged && !settings.password;

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.host')}</div>
        <Input
          aria-label={t('common.fields.host')}
          value={settings.host || ''}
          onChange={(e) => {
            onChange({ ...settings, host: e.target.value.trim() });
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="ftp.example.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.port')}</div>
        <InputNumber
          aria-label={t('common.fields.port')}
          value={settings.port}
          onChange={(value) => {
            if (!value) return;
            onChange({ ...settings, port: value });
          }}
          size="small"
          className="w-full max-w-[250px]"
          min={1}
          max={65535}
          placeholder="21"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.username')}</div>
        <Input
          aria-label={t('common.fields.username')}
          value={settings.username || ''}
          onChange={(e) => {
            onChange({ ...settings, username: e.target.value.trim() });
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storages.edit.usernamePlaceholder')}
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.password')}</div>
        <div className="w-full max-w-[250px]">
          <Input.Password
            aria-label={t('common.fields.password')}
            aria-describedby={isPasswordRequiredAgain ? passwordNoticeId : undefined}
            value={settings.password || ''}
            onChange={(e) => {
              onChange({ ...settings, password: toOptionalValue(e.target.value) });
            }}
            size="small"
            className="w-full"
            placeholder={t('storages.edit.passwordPlaceholder')}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
          {isPasswordRequiredAgain && <AddressChangedSecretNoticeComponent id={passwordNoticeId} />}
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.path')}</div>
        <div className="flex items-center">
          <Input
            aria-label={t('storages.fields.path')}
            value={settings.path || ''}
            onChange={(e) => {
              let pathValue = e.target.value.trim();

              if (pathValue.startsWith('/')) {
                pathValue = pathValue.substring(1);
              }

              onChange({ ...settings, path: pathValue || undefined });
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.optionalPlaceholder', { example: 'backups' })}
          />

          <Tooltip className="cursor-pointer" title={t('storages.edit.remotePathTooltip')}>
            <InfoCircleOutlined className="ml-2" />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.useSslTls')}</div>
        <div className="flex items-center">
          <Checkbox
            checked={settings.isSsl || false}
            onChange={(e) => {
              onChange({ ...settings, isSsl: e.target.checked });
            }}
          >
            {t('storages.edit.ftp.enableFtps')}
          </Checkbox>

          <Tooltip className="cursor-pointer" title={t('storages.edit.ftp.useSslTooltip')}>
            <InfoCircleOutlined className="ml-2" />
          </Tooltip>
        </div>
      </div>

      <AdvancedSettingsToggleComponent
        label={t('storages.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
        regionId={advancedRegionId}
      />

      {isShowAdvanced && (
        <div id={advancedRegionId}>
          {settings.isSsl && (
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
                  {t('storages.edit.ftp.skipCertificateVerification')}
                </Checkbox>

                <Tooltip
                  className="cursor-pointer"
                  title={t('storages.edit.ftp.skipTlsVerifyTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" />
                </Tooltip>
              </div>
            </div>
          )}
        </div>
      )}

      <div className="mb-5" />
    </>
  );
}
