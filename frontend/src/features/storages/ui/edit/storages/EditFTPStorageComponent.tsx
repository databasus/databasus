import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { Checkbox, Input, InputNumber, Tooltip } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { FtpStorageSettings } from '../../../../../entities/storages';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: FtpStorageSettings;
  onChange: (settings: FtpStorageSettings) => void;
}

export function EditFTPStorageComponent({ settings, onChange }: Props) {
  const { t } = useTranslation();
  const hasAdvancedValues = !!settings.isSkipTlsVerify;
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.host')}</div>
        <Input
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
        <Input.Password
          value={settings.password || ''}
          onChange={(e) => {
            onChange({ ...settings, password: toOptionalValue(e.target.value) });
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storages.edit.passwordPlaceholder')}
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.path')}</div>
        <div className="flex items-center">
          <Input
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
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
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
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            </div>
          )}
        </>
      )}

      <div className="mb-5" />
    </>
  );
}
