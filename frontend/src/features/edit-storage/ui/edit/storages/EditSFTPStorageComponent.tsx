import { Checkbox, Form, Input, InputNumber, Select } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  SFTP_AUTH_METHOD_LABEL_KEYS,
  SftpAuthMethod,
  type SftpStorageSettings,
} from '../../../../../entities/storages';
import { toOptionalValue } from '../../../../../shared/lib';
import { AdvancedSettingsToggleComponent } from '../../../../../shared/ui';

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
    <Form component="div" layout="vertical" size="small" className="w-full max-w-sm">
      <Form.Item label={t('common.fields.host')} htmlFor={`${controlId}-host`}>
        <Input
          id={`${controlId}-host`}
          value={settings.host}
          onChange={(event) => onChange({ ...settings, host: event.target.value.trim() })}
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="sftp.example.com"
        />
      </Form.Item>
      <Form.Item label={t('common.fields.port')} htmlFor={`${controlId}-port`}>
        <InputNumber
          id={`${controlId}-port`}
          value={settings.port}
          onChange={(port) => port && onChange({ ...settings, port })}
          min={1}
          max={65535}
        />
      </Form.Item>
      <Form.Item label={t('common.fields.username')} htmlFor={`${controlId}-username`}>
        <Input
          id={`${controlId}-username`}
          value={settings.username}
          onChange={(event) => onChange({ ...settings, username: event.target.value.trim() })}
          autoComplete="off"
        />
      </Form.Item>
      <Form.Item label={t('storages.fields.authMethod')} htmlFor={`${controlId}-methods`}>
        <Select
          id={`${controlId}-methods`}
          value={hasPassword && hasPrivateKey ? 'BOTH' : settings.authMethods[0]}
          options={authMethodOptions}
          virtual={false}
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
      </Form.Item>
      {hasPassword && (
        <Form.Item
          label={t('common.fields.password')}
          htmlFor={`${controlId}-password`}
          help={
            <span
              className={isPasswordSaved ? 'text-muted' : 'text-danger'}
              id={`${controlId}-password-help`}
            >
              {isPasswordSaved
                ? t('storages.edit.sftp.savedPassword')
                : !settings.password
                  ? missingSecretText('requiredPassword')
                  : undefined}
            </span>
          }
          validateStatus={!settings.password && !isPasswordSaved ? 'error' : undefined}
        >
          <Input.Password
            id={`${controlId}-password`}
            aria-describedby={`${controlId}-password-help`}
            value={settings.password ?? ''}
            onChange={(event) =>
              onChange({ ...settings, password: toOptionalValue(event.target.value) })
            }
            autoComplete="new-password"
          />
        </Form.Item>
      )}
      {hasPrivateKey && (
        <Form.Item
          label={t('storages.fields.privateKey')}
          htmlFor={`${controlId}-key`}
          help={
            <span
              className={isPrivateKeySaved ? 'text-muted' : 'text-danger'}
              id={`${controlId}-key-help`}
            >
              {isPrivateKeySaved
                ? t('storages.edit.sftp.savedPrivateKey')
                : !settings.privateKey
                  ? missingSecretText('requiredPrivateKey')
                  : t('storages.edit.sftp.privateKeyTooltip')}
            </span>
          }
          validateStatus={!settings.privateKey && !isPrivateKeySaved ? 'error' : undefined}
        >
          <Input.TextArea
            id={`${controlId}-key`}
            aria-describedby={`${controlId}-key-help`}
            value={settings.privateKey ?? ''}
            onChange={(event) =>
              onChange({ ...settings, privateKey: toOptionalValue(event.target.value) })
            }
            rows={4}
          />
        </Form.Item>
      )}
      <Form.Item label={t('storages.fields.path')} htmlFor={`${controlId}-path`}>
        <Input
          id={`${controlId}-path`}
          value={settings.path ?? ''}
          onChange={(event) => onChange({ ...settings, path: toOptionalValue(event.target.value) })}
        />
      </Form.Item>
      <AdvancedSettingsToggleComponent
        label={t('storages.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
        regionId={`${controlId}-advanced`}
      />
      <div id={`${controlId}-advanced`} hidden={!isShowAdvanced}>
        <Form.Item
          help={
            <span id={`${controlId}-skip-host-key-help`}>
              {t('storages.edit.sftp.skipHostKeyTooltip')}
            </span>
          }
        >
          <Checkbox
            aria-describedby={`${controlId}-skip-host-key-help`}
            checked={settings.isSkipHostKeyVerify}
            onChange={(event) =>
              onChange({ ...settings, isSkipHostKeyVerify: event.target.checked })
            }
          >
            {t('storages.edit.sftp.skipHostKeyVerification')}
          </Checkbox>
        </Form.Item>
      </div>
    </Form>
  );
}
