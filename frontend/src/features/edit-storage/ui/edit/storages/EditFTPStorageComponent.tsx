import { Checkbox, NumberInput, PasswordInput, TextInput, Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { FtpStorageSettings } from '@/entities/storages';
import { toOptionalValue } from '@/shared/lib';
import { AddressChangedSecretNoticeComponent, AdvancedSettingsToggleComponent } from '@/shared/ui';

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
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
          {t('common.fields.host')}
        </div>
        <TextInput
          aria-label={t('common.fields.host')}
          value={settings.host || ''}
          onChange={(e) => {
            onChange({ ...settings, host: e.target.value.trim() });
          }}
          size="xs"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="ftp.example.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
          {t('common.fields.port')}
        </div>
        <NumberInput
          aria-label={t('common.fields.port')}
          value={settings.port}
          onChange={(value) => {
            if (typeof value !== 'number' || !value) return;
            onChange({ ...settings, port: value });
          }}
          size="xs"
          className="w-full max-w-[250px]"
          min={1}
          max={65535}
          placeholder="21"
          role="spinbutton"
          aria-valuenow={typeof settings.port === 'number' ? settings.port : undefined}
          aria-valuemin={1}
          aria-valuemax={65535}
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
          {t('common.fields.username')}
        </div>
        <TextInput
          aria-label={t('common.fields.username')}
          value={settings.username || ''}
          onChange={(e) => {
            onChange({ ...settings, username: e.target.value.trim() });
          }}
          size="xs"
          className="w-full max-w-[250px]"
          placeholder={t('storages.edit.usernamePlaceholder')}
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
          {t('common.fields.password')}
        </div>
        <div className="w-full max-w-[250px]">
          <PasswordInput
            aria-label={t('common.fields.password')}
            value={settings.password || ''}
            onChange={(e) => {
              onChange({ ...settings, password: toOptionalValue(e.target.value) });
            }}
            size="xs"
            className="w-full"
            placeholder={t('storages.edit.passwordPlaceholder')}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
            visibilityToggleButtonProps={{
              'aria-label': t('common.actions.togglePasswordVisibility'),
            }}
            description={
              isPasswordRequiredAgain && (
                <AddressChangedSecretNoticeComponent id={passwordNoticeId} />
              )
            }
          />
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
          {t('storages.fields.path')}
        </div>
        <div className="flex w-full min-w-0 items-center">
          <TextInput
            aria-label={t('storages.fields.path')}
            value={settings.path || ''}
            onChange={(e) => {
              let pathValue = e.target.value.trim();

              if (pathValue.startsWith('/')) {
                pathValue = pathValue.substring(1);
              }

              onChange({ ...settings, path: pathValue || undefined });
            }}
            size="xs"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.optionalPlaceholder', { example: 'backups' })}
          />

          <Tooltip className="cursor-pointer" label={t('storages.edit.remotePathTooltip')}>
            <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
          {t('storages.fields.useSslTls')}
        </div>
        <div className="flex items-center">
          <Checkbox
            checked={settings.isSsl || false}
            onChange={(e) => {
              onChange({ ...settings, isSsl: e.target.checked });
            }}
            label={<>{t('storages.edit.ftp.enableFtps')}</>}
          />

          <Tooltip className="cursor-pointer" label={t('storages.edit.ftp.useSslTooltip')}>
            <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
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
              <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
                {t('storages.fields.skipTlsVerify')}
              </div>
              <div className="flex items-center">
                <Checkbox
                  checked={settings.isSkipTlsVerify || false}
                  onChange={(e) => {
                    onChange({ ...settings, isSkipTlsVerify: e.target.checked });
                  }}
                  label={<>{t('storages.edit.ftp.skipCertificateVerification')}</>}
                />

                <Tooltip
                  className="cursor-pointer"
                  label={t('storages.edit.ftp.skipTlsVerifyTooltip')}
                >
                  <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
                </Tooltip>
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}
