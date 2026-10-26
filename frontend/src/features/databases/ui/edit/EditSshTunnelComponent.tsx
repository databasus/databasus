import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Checkbox, Input, InputNumber, Select, Tooltip } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  DEFAULT_SSH_PORT,
  SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS,
  SshTunnelAuthType,
  type SshTunnelSettings,
  createEmptySshTunnelSettings,
  setSshTunnelAuthTypeAndClearUnusedSecrets,
} from '../../../../entity/databases';

interface Props {
  sshTunnel: SshTunnelSettings;
  hasStoredSecrets: boolean;
  onChange: (sshTunnel: SshTunnelSettings) => void;
}

export const EditSshTunnelComponent = ({ sshTunnel, hasStoredSecrets, onChange }: Props) => {
  const { t } = useTranslation();
  const [isReplacingSecrets, setIsReplacingSecrets] = useState(false);

  const updateField = <Field extends keyof SshTunnelSettings>(
    field: Field,
    value: SshTunnelSettings[Field],
  ) => {
    onChange({ ...currentTunnel, [field]: value });
  };

  const startReplacingSecrets = () => {
    setIsReplacingSecrets(true);
    onChange({ ...currentTunnel, password: '', privateKey: '', privateKeyPassphrase: '' });
  };

  const changeAuthType = (authType: SshTunnelAuthType) => {
    onChange(setSshTunnelAuthTypeAndClearUnusedSecrets(currentTunnel, authType));
  };

  const renderStoredSecrets = () => (
    <div className="mb-3 flex w-full items-center">
      <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.credentials')}</div>
      <div className="flex items-center">
        <span className="mr-3">*************</span>
        <Button size="small" onClick={startReplacingSecrets}>
          {t('databases.actions.replace')}
        </Button>
      </div>
    </div>
  );

  const renderPassword = () => (
    <div className="mb-3 flex w-full items-center">
      <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.password')}</div>
      <Input.Password
        value={currentTunnel.password}
        onChange={(e) => updateField('password', e.target.value)}
        size="small"
        className="max-w-[200px] grow"
        placeholder={t('databases.sshTunnel.passwordPlaceholder')}
        autoComplete="off"
        data-1p-ignore
        data-lpignore="true"
        data-form-type="other"
      />
    </div>
  );

  const renderPrivateKey = () => (
    <>
      <div className="mb-1 flex w-full items-start">
        <div className="min-w-[150px] pr-2 leading-6">{t('databases.sshTunnel.privateKey')}</div>
        <Input.TextArea
          value={currentTunnel.privateKey}
          onChange={(e) => updateField('privateKey', e.target.value)}
          size="small"
          className="max-w-[200px] grow"
          // eslint-disable-next-line i18next/no-literal-string -- key format example, not copy
          placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
          autoSize={{ minRows: 2, maxRows: 5 }}
        />
      </div>

      <div className="mb-3 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.keyPassphrase')}</div>
        <Input.Password
          value={currentTunnel.privateKeyPassphrase}
          onChange={(e) => updateField('privateKeyPassphrase', e.target.value)}
          size="small"
          className="max-w-[200px] grow"
          placeholder={t('databases.sshTunnel.keyPassphrasePlaceholder')}
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
        />
      </div>
    </>
  );

  const renderCredentials = () => {
    if (hasStoredSecrets && !isReplacingSecrets) return renderStoredSecrets();

    return currentTunnel.authType === SshTunnelAuthType.PRIVATE_KEY
      ? renderPrivateKey()
      : renderPassword();
  };

  // A saved database without a tunnel comes back as only the switch, so enabling it starts from
  // the defaults.
  const currentTunnel = { ...createEmptySshTunnelSettings(), ...sshTunnel };

  return (
    <>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.title')}</div>
        <Checkbox
          checked={currentTunnel.isEnabled}
          onChange={(e) => updateField('isEnabled', e.target.checked)}
        >
          <Tooltip className="cursor-pointer" title={t('databases.sshTunnel.tooltip')}>
            <InfoCircleOutlined style={{ color: 'gray' }} />
          </Tooltip>
        </Checkbox>
      </div>

      {currentTunnel.isEnabled && (
        <>
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.host')}</div>
            <Input
              value={currentTunnel.host}
              onChange={(e) => updateField('host', e.target.value)}
              size="small"
              className="max-w-[200px] grow"
              // eslint-disable-next-line i18next/no-literal-string -- example hostname
              placeholder="bastion.example.com"
            />

            <Tooltip className="cursor-pointer" title={t('databases.sshTunnel.hostTooltip')}>
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.port')}</div>
            <InputNumber
              value={currentTunnel.port}
              onChange={(value) => updateField('port', value ?? DEFAULT_SSH_PORT)}
              size="small"
              className="max-w-[200px] grow"
              min={1}
              max={65535}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.username')}</div>
            <Input
              value={currentTunnel.username}
              onChange={(e) => updateField('username', e.target.value)}
              size="small"
              className="max-w-[200px] grow"
              placeholder={t('databases.sshTunnel.usernamePlaceholder')}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.authType')}</div>
            <Select
              value={currentTunnel.authType}
              onChange={changeAuthType}
              options={Object.values(SshTunnelAuthType).map((authType) => ({
                label: t(SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS[authType]),
                value: authType,
              }))}
              size="small"
              className="max-w-[200px] grow"
            />
          </div>

          {renderCredentials()}
        </>
      )}
    </>
  );
};
