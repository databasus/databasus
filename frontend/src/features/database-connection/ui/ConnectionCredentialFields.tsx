import { Input, InputNumber } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { AddressChangedSecretNoticeComponent } from '@/shared/ui';

import { ConnectionFieldRow } from './ConnectionFieldRow';

interface CredentialValues {
  host: string;
  port?: number;
  username: string;
  password?: string;
  database?: string;
}

interface Props {
  controlId: string;
  engineName: string;
  values: CredentialValues;
  isAddressChanged: boolean;
  isPortShown?: boolean;
  isDatabaseNameShown?: boolean;
  portPlaceholder?: string;
  hostHints?: ReactNode;
  onHostChange: (rawHost: string) => void;
  onValuesChange: (change: Partial<CredentialValues>) => void;
}

export const ConnectionCredentialFields = ({
  controlId,
  engineName,
  values,
  isAddressChanged,
  isPortShown = true,
  isDatabaseNameShown = true,
  portPlaceholder,
  hostHints,
  onHostChange,
  onValuesChange,
}: Props) => {
  const { t } = useTranslation();
  const isPasswordRequiredAgain = isAddressChanged && !values.password?.trim();

  return (
    <>
      <ConnectionFieldRow label={t('common.fields.host')} controlId={`${controlId}-host`}>
        <Input
          id={`${controlId}-host`}
          value={values.host}
          onChange={(e) => onHostChange(e.target.value)}
          size="small"
          className="w-full max-w-[200px]"
          placeholder={t('databases.edit.placeholders.host', { engine: engineName })}
        />
      </ConnectionFieldRow>

      {hostHints}

      {isPortShown && (
        <ConnectionFieldRow label={t('common.fields.port')} controlId={`${controlId}-port`}>
          <InputNumber
            id={`${controlId}-port`}
            type="number"
            value={values.port}
            onChange={(port) => {
              if (port === null) return;

              onValuesChange({ port });
            }}
            size="small"
            className="w-full max-w-[200px]"
            placeholder={
              portPlaceholder ?? t('databases.edit.placeholders.port', { engine: engineName })
            }
          />
        </ConnectionFieldRow>
      )}

      <ConnectionFieldRow label={t('common.fields.username')} controlId={`${controlId}-username`}>
        <Input
          id={`${controlId}-username`}
          value={values.username}
          onChange={(e) => onValuesChange({ username: e.target.value.trim() })}
          size="small"
          className="w-full max-w-[200px]"
          placeholder={t('databases.edit.placeholders.username', { engine: engineName })}
        />
      </ConnectionFieldRow>

      <ConnectionFieldRow label={t('common.fields.password')} controlId={`${controlId}-password`}>
        <div className="w-full max-w-[200px]">
          <Input.Password
            id={`${controlId}-password`}
            aria-describedby={isPasswordRequiredAgain ? `${controlId}-password-notice` : undefined}
            value={values.password}
            onChange={(e) => onValuesChange({ password: e.target.value })}
            size="small"
            className="w-full"
            placeholder={t('databases.edit.placeholders.password', { engine: engineName })}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
          {isPasswordRequiredAgain && (
            <AddressChangedSecretNoticeComponent id={`${controlId}-password-notice`} />
          )}
        </div>
      </ConnectionFieldRow>

      {isDatabaseNameShown && (
        <ConnectionFieldRow
          label={t('databases.fields.databaseName')}
          controlId={`${controlId}-database`}
        >
          <Input
            id={`${controlId}-database`}
            value={values.database}
            onChange={(e) => onValuesChange({ database: e.target.value.trim() })}
            size="small"
            className="w-full max-w-[200px]"
            placeholder={t('databases.edit.placeholders.databaseName', { engine: engineName })}
          />
        </ConnectionFieldRow>
      )}
    </>
  );
};
