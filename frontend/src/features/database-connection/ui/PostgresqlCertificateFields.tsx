import { Button, Textarea } from '@mantine/core';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import { PostgresSslMode } from '@/entities/databases';
import { ConnectionFieldRow } from '@/features/database-connection/ui/ConnectionFieldRow';
import { ConnectionFieldTooltip } from '@/features/database-connection/ui/ConnectionFieldTooltip';
import { AddressChangedSecretNoticeComponent } from '@/shared/ui';

interface CertificateValues {
  sslMode?: PostgresSslMode;
  sslClientCert?: string;
  sslClientKey?: string;
  sslRootCert?: string;
}

interface Props {
  values: CertificateValues;
  savedClientCertificate?: string;
  isAddressChanged: boolean;
  onChange: (
    change: Pick<CertificateValues, 'sslClientCert' | 'sslClientKey' | 'sslRootCert'>,
  ) => void;
}

// eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
const CERTIFICATE_PLACEHOLDER = '-----BEGIN CERTIFICATE-----';
// eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
const PRIVATE_KEY_PLACEHOLDER = '-----BEGIN PRIVATE KEY-----';

export const PostgresqlCertificateFields = ({
  values,
  savedClientCertificate,
  isAddressChanged,
  onChange,
}: Props) => {
  const { t } = useTranslation();
  const controlId = useId();

  if ((values.sslMode ?? PostgresSslMode.Disable) === PostgresSslMode.Disable) return null;

  // The mask stands for the saved certificate only while the form still holds it, because a save
  // sends what the form holds.
  const isCertificateStored =
    !!savedClientCertificate && values.sslClientCert === savedClientCertificate;
  const isClientKeyRequiredAgain =
    isAddressChanged && !!values.sslClientCert && !values.sslClientKey;

  const renderClientKey = () => (
    <ConnectionFieldRow
      label={t('databases.fields.clientKey')}
      controlId={`${controlId}-client-key`}
      isTopAligned
    >
      <div className="w-full max-w-[300px]">
        <Textarea
          id={`${controlId}-client-key`}
          value={values.sslClientKey || ''}
          onChange={(e) => onChange({ sslClientKey: e.target.value })}
          size="xs"
          className="w-full"
          placeholder={PRIVATE_KEY_PLACEHOLDER}
          description={
            isClientKeyRequiredAgain && (
              <AddressChangedSecretNoticeComponent id={`${controlId}-client-key-notice`} />
            )
          }
        />
      </div>
    </ConnectionFieldRow>
  );

  if (isCertificateStored) {
    return (
      <>
        <ConnectionFieldRow label={t('databases.fields.clientCertificate')}>
          <div className="flex items-center">
            <span className="mr-3">*************</span>
            <Button
              size="xs"
              onClick={() => onChange({ sslClientCert: '', sslClientKey: '', sslRootCert: '' })}
            >
              {t('databases.actions.replace')}
            </Button>
          </div>
        </ConnectionFieldRow>

        {(isAddressChanged || !!values.sslClientKey) && renderClientKey()}
      </>
    );
  }

  return (
    <>
      <ConnectionFieldRow
        label={t('databases.fields.clientCertificate')}
        controlId={`${controlId}-client-cert`}
        isTopAligned
      >
        <Textarea
          id={`${controlId}-client-cert`}
          value={values.sslClientCert || ''}
          onChange={(e) => onChange({ sslClientCert: e.target.value })}
          size="xs"
          className="w-full max-w-[300px]"
          placeholder={CERTIFICATE_PLACEHOLDER}
        />
      </ConnectionFieldRow>

      {renderClientKey()}

      <ConnectionFieldRow
        label={t('databases.fields.serverCaCertificate')}
        controlId={`${controlId}-root-cert`}
        isTopAligned
      >
        <div className="flex w-full items-start">
          <Textarea
            id={`${controlId}-root-cert`}
            value={values.sslRootCert || ''}
            onChange={(e) => onChange({ sslRootCert: e.target.value })}
            size="xs"
            className="w-full max-w-[300px]"
            placeholder={CERTIFICATE_PLACEHOLDER}
          />

          <ConnectionFieldTooltip title={t('databases.edit.serverCaCertificateTooltip')} />
        </div>
      </ConnectionFieldRow>
    </>
  );
};
