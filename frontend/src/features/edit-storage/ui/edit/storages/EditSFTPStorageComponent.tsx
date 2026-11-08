import { Checkbox, NumberInput, PasswordInput, TextInput, Textarea } from '@mantine/core';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  SFTP_AUTH_METHOD_LABEL_KEYS,
  SftpAuthMethod,
  type SftpStorageSettings,
} from '@/entities/storages';
import { toOptionalValue } from '@/shared/lib';
import { WrappingSelect } from '@/shared/ui';
import { AdvancedSettingsToggleComponent } from '@/shared/ui';

interface Props {
  settings: SftpStorageSettings;
  savedAuthMethods: SftpAuthMethod[];
  isAddressChanged: boolean;
  onChange: (settings: SftpStorageSettings) => void;
}

export function EditSFTPStorageComponent({
  settings,
  savedAuthMethods,
  isAddressChanged,
  onChange,
}: Props) {
  const [isShowAdvanced, setIsShowAdvanced] = useState(settings.isSkipHostKeyVerify);
  const { t } = useTranslation();
  const controlId = useId();
  const hasPassword = settings.authMethods.includes(SftpAuthMethod.PASSWORD);
  const hasPrivateKey = settings.authMethods.includes(SftpAuthMethod.PRIVATE_KEY);
  const isPasswordSaved = !isAddressChanged && savedAuthMethods.includes(SftpAuthMethod.PASSWORD);
  const isPrivateKeySaved =
    !isAddressChanged && savedAuthMethods.includes(SftpAuthMethod.PRIVATE_KEY);
  const missingSecretText = (requiredKey: 'requiredPassword' | 'requiredPrivateKey') =>
    isAddressChanged
      ? t('common.messages.addressChangedEnterSecretAgain')
      : t(`storages.edit.sftp.${requiredKey}`);
  const authMethodOptions = [
    { value: SftpAuthMethod.PASSWORD, label: t(SFTP_AUTH_METHOD_LABEL_KEYS.PASSWORD) },
    { value: SftpAuthMethod.PRIVATE_KEY, label: t(SFTP_AUTH_METHOD_LABEL_KEYS.PRIVATE_KEY) },
    { value: 'BOTH', label: t('storages.sftpAuthMethods.both') },
  ];

  return (
    <div className="w-full max-w-sm">
      <div className="mb-3 min-w-0">
        <label htmlFor={`${controlId}-host`} className="mb-1 block text-sm text-content">
          {t('common.fields.host')}
        </label>
        <TextInput
          id={`${controlId}-host`}
          value={settings.host}
          onChange={(event) => onChange({ ...settings, host: event.target.value.trim() })}
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="sftp.example.com"
        />
      </div>
      <div className="mb-3 min-w-0">
        <label htmlFor={`${controlId}-port`} className="mb-1 block text-sm text-content">
          {t('common.fields.port')}
        </label>
        <NumberInput
          id={`${controlId}-port`}
          value={settings.port}
          onChange={(port) => {
            if (typeof port === 'number' && port) onChange({ ...settings, port });
          }}
          min={1}
          max={65535}
          role="spinbutton"
          aria-valuenow={typeof settings.port === 'number' ? settings.port : undefined}
          aria-valuemin={1}
          aria-valuemax={65535}
        />
      </div>
      <div className="mb-3 min-w-0">
        <label htmlFor={`${controlId}-username`} className="mb-1 block text-sm text-content">
          {t('common.fields.username')}
        </label>
        <TextInput
          id={`${controlId}-username`}
          value={settings.username}
          onChange={(event) => onChange({ ...settings, username: event.target.value.trim() })}
          autoComplete="off"
        />
      </div>
      <div className="mb-3 min-w-0">
        <label htmlFor={`${controlId}-methods`} className="mb-1 block text-sm text-content">
          {t('storages.fields.authMethod')}
        </label>
        <WrappingSelect
          id={`${controlId}-methods`}
          value={hasPassword && hasPrivateKey ? 'BOTH' : settings.authMethods[0]}
          data={authMethodOptions}
          onChange={(method) => {
            const authMethods: SftpAuthMethod[] =
              method === 'BOTH'
                ? [SftpAuthMethod.PASSWORD, SftpAuthMethod.PRIVATE_KEY]
                : [method as SftpAuthMethod];
            onChange({
              ...settings,
              authMethods,
              password: authMethods.includes(SftpAuthMethod.PASSWORD)
                ? settings.password
                : undefined,
              privateKey: authMethods.includes(SftpAuthMethod.PRIVATE_KEY)
                ? settings.privateKey
                : undefined,
            });
          }}
        />
      </div>
      {hasPassword && (
        <div className="mb-3 min-w-0">
          <label htmlFor={`${controlId}-password`} className="mb-1 block text-sm text-content">
            {t('common.fields.password')}
          </label>
          <PasswordInput
            id={`${controlId}-password`}
            value={settings.password ?? ''}
            onChange={(event) =>
              onChange({ ...settings, password: toOptionalValue(event.target.value) })
            }
            autoComplete="new-password"
            visibilityToggleButtonProps={{
              'aria-label': t('common.actions.togglePasswordVisibility'),
            }}
            aria-invalid={!settings.password && !isPasswordSaved ? true : undefined}
            inputWrapperOrder={['label', 'input', 'description', 'error']}
            description={isPasswordSaved ? t('storages.edit.sftp.savedPassword') : undefined}
            error={
              !settings.password && !isPasswordSaved
                ? missingSecretText('requiredPassword')
                : undefined
            }
          />
        </div>
      )}
      {hasPrivateKey && (
        <div className="mb-3 min-w-0">
          <label htmlFor={`${controlId}-key`} className="mb-1 block text-sm text-content">
            {t('storages.fields.privateKey')}
          </label>
          <Textarea
            id={`${controlId}-key`}
            value={settings.privateKey ?? ''}
            onChange={(event) =>
              onChange({ ...settings, privateKey: toOptionalValue(event.target.value) })
            }
            rows={4}
            inputWrapperOrder={['label', 'input', 'description', 'error']}
            description={
              isPrivateKeySaved
                ? t('storages.edit.sftp.savedPrivateKey')
                : settings.privateKey
                  ? t('storages.edit.sftp.privateKeyTooltip')
                  : undefined
            }
            descriptionProps={{ className: isPrivateKeySaved ? 'text-muted' : 'text-danger' }}
            error={
              !settings.privateKey && !isPrivateKeySaved
                ? missingSecretText('requiredPrivateKey')
                : undefined
            }
          />
        </div>
      )}
      <div className="mb-3 min-w-0">
        <label htmlFor={`${controlId}-path`} className="mb-1 block text-sm text-content">
          {t('storages.fields.path')}
        </label>
        <TextInput
          id={`${controlId}-path`}
          value={settings.path ?? ''}
          onChange={(event) => onChange({ ...settings, path: toOptionalValue(event.target.value) })}
        />
      </div>
      <AdvancedSettingsToggleComponent
        label={t('storages.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
        regionId={`${controlId}-advanced`}
      />
      <div id={`${controlId}-advanced`} hidden={!isShowAdvanced}>
        <div className="mb-3 min-w-0">
          <Checkbox
            aria-describedby={`${controlId}-skip-host-key-help`}
            checked={settings.isSkipHostKeyVerify}
            onChange={(event) =>
              onChange({ ...settings, isSkipHostKeyVerify: event.target.checked })
            }
            label={<>{t('storages.edit.sftp.skipHostKeyVerification')}</>}
          />
          <div className="text-sm text-danger">
            <span id={`${controlId}-skip-host-key-help`}>
              {t('storages.edit.sftp.skipHostKeyTooltip')}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
