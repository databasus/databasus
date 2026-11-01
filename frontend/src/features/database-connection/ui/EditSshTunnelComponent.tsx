import { Button, Checkbox, Input, InputNumber, Select } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  DEFAULT_SSH_PORT,
  SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS,
  SshTunnelAuthType,
  type SshTunnelSettings,
  createEmptySshTunnelSettings,
  hasStoredSshTunnelSecretsForAuthType,
  setSshTunnelAuthTypeAndClearUnusedSecrets,
} from '../../../entities/databases';
import { ConnectionFieldRow } from './ConnectionFieldRow';
import { ConnectionFieldTooltip } from './ConnectionFieldTooltip';

interface Props {
  sshTunnel: SshTunnelSettings;
  savedSshTunnel?: SshTunnelSettings;
  onChange: (sshTunnel: SshTunnelSettings) => void;
}

export const EditSshTunnelComponent = ({ sshTunnel, savedSshTunnel, onChange }: Props) => {
  const { t } = useTranslation();
  const controlId = useId();
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
    <ConnectionFieldRow label={t('databases.sshTunnel.credentials')}>
      <div className="mb-2 flex items-center">
        <span className="mr-3">*************</span>
        <Button size="small" onClick={startReplacingSecrets}>
          {t('databases.actions.replace')}
        </Button>
      </div>
    </ConnectionFieldRow>
  );

  const renderPassword = () => (
    <ConnectionFieldRow
      label={t('databases.sshTunnel.password')}
      controlId={`${controlId}-password`}
    >
      <Input.Password
        id={`${controlId}-password`}
        value={currentTunnel.password}
        onChange={(e) => updateField('password', e.target.value)}
        size="small"
        className="mb-2 w-full max-w-[200px]"
        placeholder={t('databases.sshTunnel.passwordPlaceholder')}
        autoComplete="off"
        data-1p-ignore
        data-lpignore="true"
        data-form-type="other"
      />
    </ConnectionFieldRow>
  );

  const renderPrivateKey = () => (
    <>
      <ConnectionFieldRow
        label={t('databases.sshTunnel.privateKey')}
        controlId={`${controlId}-private-key`}
        isTopAligned
      >
        <Input.TextArea
          id={`${controlId}-private-key`}
          value={currentTunnel.privateKey}
          onChange={(e) => updateField('privateKey', e.target.value)}
          size="small"
          className="w-full max-w-[200px]"
          // eslint-disable-next-line i18next/no-literal-string -- key format example, not copy
          placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
          autoSize={{ minRows: 2, maxRows: 5 }}
        />
      </ConnectionFieldRow>

      <ConnectionFieldRow
        label={t('databases.sshTunnel.keyPassphrase')}
        controlId={`${controlId}-passphrase`}
      >
        <Input.Password
          id={`${controlId}-passphrase`}
          value={currentTunnel.privateKeyPassphrase}
          onChange={(e) => updateField('privateKeyPassphrase', e.target.value)}
          size="small"
          className="mb-2 w-full max-w-[200px]"
          placeholder={t('databases.sshTunnel.keyPassphrasePlaceholder')}
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
        />
      </ConnectionFieldRow>
    </>
  );

  const renderCredentials = () => {
    if (hasStoredSecrets && !isReplacingSecrets && !hasTypedSecrets) return renderStoredSecrets();

    return currentTunnel.authType === SshTunnelAuthType.PRIVATE_KEY
      ? renderPrivateKey()
      : renderPassword();
  };

  // A saved database without a tunnel comes back as only the switch, so enabling it starts from
  // the defaults.
  const currentTunnel = { ...createEmptySshTunnelSettings(), ...sshTunnel };
  const hasStoredSecrets = hasStoredSshTunnelSecretsForAuthType(
    savedSshTunnel,
    currentTunnel.authType,
  );
  // This component leaves the screen with the advanced settings, so the typed value itself tells
  // that a replacement is under way when it comes back.
  const hasTypedSecrets =
    !!currentTunnel.password || !!currentTunnel.privateKey || !!currentTunnel.privateKeyPassphrase;

  return (
    <>
      <ConnectionFieldRow label={t('databases.sshTunnel.title')} controlId={`${controlId}-enabled`}>
        <div className="flex items-center">
          <Checkbox
            id={`${controlId}-enabled`}
            checked={currentTunnel.isEnabled}
            onChange={(e) => updateField('isEnabled', e.target.checked)}
          />

          <ConnectionFieldTooltip title={t('databases.sshTunnel.tooltip')} />
        </div>
      </ConnectionFieldRow>

      {currentTunnel.isEnabled && (
        <>
          <ConnectionFieldRow label={t('databases.sshTunnel.host')} controlId={`${controlId}-host`}>
            <div className="flex w-full items-center">
              <Input
                id={`${controlId}-host`}
                value={currentTunnel.host}
                onChange={(e) => updateField('host', e.target.value)}
                size="small"
                className="w-full max-w-[200px]"
                // eslint-disable-next-line i18next/no-literal-string -- example hostname
                placeholder="bastion.example.com"
              />

              <ConnectionFieldTooltip title={t('databases.sshTunnel.hostTooltip')} />
            </div>
          </ConnectionFieldRow>

          <ConnectionFieldRow label={t('databases.sshTunnel.port')} controlId={`${controlId}-port`}>
            <InputNumber
              id={`${controlId}-port`}
              value={currentTunnel.port}
              onChange={(value) => updateField('port', value ?? DEFAULT_SSH_PORT)}
              size="small"
              className="w-full max-w-[200px]"
              min={1}
              max={65535}
            />
          </ConnectionFieldRow>

          <ConnectionFieldRow
            label={t('databases.sshTunnel.username')}
            controlId={`${controlId}-username`}
          >
            <Input
              id={`${controlId}-username`}
              value={currentTunnel.username}
              onChange={(e) => updateField('username', e.target.value)}
              size="small"
              className="w-full max-w-[200px]"
              placeholder={t('databases.sshTunnel.usernamePlaceholder')}
            />
          </ConnectionFieldRow>

          <ConnectionFieldRow
            label={t('databases.sshTunnel.authType')}
            controlId={`${controlId}-auth-type`}
          >
            <Select
              id={`${controlId}-auth-type`}
              value={currentTunnel.authType}
              onChange={changeAuthType}
              options={Object.values(SshTunnelAuthType).map((authType) => ({
                label: t(SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS[authType]),
                value: authType,
              }))}
              size="small"
              className="w-full max-w-[200px]"
            />
          </ConnectionFieldRow>

          {renderCredentials()}
        </>
      )}
    </>
  );
};
