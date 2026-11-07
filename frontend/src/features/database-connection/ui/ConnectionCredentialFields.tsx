import { NumberInput, PasswordInput, TextInput } from '@mantine/core';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { ConnectionFieldRow } from '@/features/database-connection/ui/ConnectionFieldRow';
import { AddressChangedSecretNoticeComponent } from '@/shared/ui';

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
        <TextInput
          id={`${controlId}-host`}
          value={values.host}
          onChange={(e) => onHostChange(e.target.value)}
          size="xs"
          className="w-full max-w-[200px]"
          placeholder={t('databases.edit.placeholders.host', { engine: engineName })}
        />
      </ConnectionFieldRow>

      {hostHints}

      {isPortShown && (
        <ConnectionFieldRow label={t('common.fields.port')} controlId={`${controlId}-port`}>
          <NumberInput
            id={`${controlId}-port`}
            value={values.port}
            onChange={(port) => {
              if (typeof port !== 'number') return;

              onValuesChange({ port });
            }}
            size="xs"
            className="w-full max-w-[200px]"
            placeholder={
              portPlaceholder ?? t('databases.edit.placeholders.port', { engine: engineName })
            }
            role="spinbutton"
            aria-valuenow={typeof values.port === 'number' ? values.port : undefined}
          />
        </ConnectionFieldRow>
      )}

      <ConnectionFieldRow label={t('common.fields.username')} controlId={`${controlId}-username`}>
        <TextInput
          id={`${controlId}-username`}
          value={values.username}
          onChange={(e) => onValuesChange({ username: e.target.value.trim() })}
          size="xs"
          className="w-full max-w-[200px]"
          placeholder={t('databases.edit.placeholders.username', { engine: engineName })}
        />
      </ConnectionFieldRow>

      <ConnectionFieldRow label={t('common.fields.password')} controlId={`${controlId}-password`}>
        <div className="w-full max-w-[200px]">
          <PasswordInput
            id={`${controlId}-password`}
            value={values.password ?? ''}
            onChange={(e) => onValuesChange({ password: e.target.value })}
            size="xs"
            className="w-full"
            placeholder={t('databases.edit.placeholders.password', { engine: engineName })}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
            description={
              isPasswordRequiredAgain && (
                <AddressChangedSecretNoticeComponent id={`${controlId}-password-notice`} />
              )
            }
          />
        </div>
      </ConnectionFieldRow>

      {isDatabaseNameShown && (
        <ConnectionFieldRow
          label={t('databases.fields.databaseName')}
          controlId={`${controlId}-database`}
        >
          <TextInput
            id={`${controlId}-database`}
            value={values.database}
            onChange={(e) => onValuesChange({ database: e.target.value.trim() })}
            size="xs"
            className="w-full max-w-[200px]"
            placeholder={t('databases.edit.placeholders.databaseName', { engine: engineName })}
          />
        </ConnectionFieldRow>
      )}
    </>
  );
};
