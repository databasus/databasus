import { Button, Input } from 'antd';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import { PostgresSslMode } from '@/entities/databases';

import { ConnectionFieldRow } from './ConnectionFieldRow';
import { ConnectionFieldTooltip } from './ConnectionFieldTooltip';

interface CertificateValues {
  sslMode?: PostgresSslMode;
  sslClientCert?: string;
  sslClientKey?: string;
  sslRootCert?: string;
}

interface Props {
  values: CertificateValues;
  savedClientCertificate?: string;
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
  onChange,
}: Props) => {
  const { t } = useTranslation();
  const controlId = useId();

  if ((values.sslMode ?? PostgresSslMode.Disable) === PostgresSslMode.Disable) return null;

  // The mask stands for the saved certificate only while the form still holds it, because a save
  // sends what the form holds.
  const isCertificateStored =
    !!savedClientCertificate && values.sslClientCert === savedClientCertificate;

  if (isCertificateStored) {
    return (
      <ConnectionFieldRow label={t('databases.fields.clientCertificate')}>
        <div className="flex items-center">
          <span className="mr-3">*************</span>
          <Button
            size="small"
            onClick={() => onChange({ sslClientCert: '', sslClientKey: '', sslRootCert: '' })}
          >
            {t('databases.actions.replace')}
          </Button>
        </div>
      </ConnectionFieldRow>
    );
  }

  return (
    <>
      <ConnectionFieldRow
        label={t('databases.fields.clientCertificate')}
        controlId={`${controlId}-client-cert`}
        isTopAligned
      >
        <Input.TextArea
          id={`${controlId}-client-cert`}
          value={values.sslClientCert || ''}
          onChange={(e) => onChange({ sslClientCert: e.target.value })}
          size="small"
          className="w-full max-w-[300px]"
          placeholder={CERTIFICATE_PLACEHOLDER}
          autoSize={{ minRows: 2, maxRows: 5 }}
        />
      </ConnectionFieldRow>

      <ConnectionFieldRow
        label={t('databases.fields.clientKey')}
        controlId={`${controlId}-client-key`}
        isTopAligned
      >
        <Input.TextArea
          id={`${controlId}-client-key`}
          value={values.sslClientKey || ''}
          onChange={(e) => onChange({ sslClientKey: e.target.value })}
          size="small"
          className="w-full max-w-[300px]"
          placeholder={PRIVATE_KEY_PLACEHOLDER}
          autoSize={{ minRows: 2, maxRows: 5 }}
        />
      </ConnectionFieldRow>

      <ConnectionFieldRow
        label={t('databases.fields.serverCaCertificate')}
        controlId={`${controlId}-root-cert`}
        isTopAligned
      >
        <div className="flex w-full items-start">
          <Input.TextArea
            id={`${controlId}-root-cert`}
            value={values.sslRootCert || ''}
            onChange={(e) => onChange({ sslRootCert: e.target.value })}
            size="small"
            className="w-full max-w-[300px]"
            placeholder={CERTIFICATE_PLACEHOLDER}
            autoSize={{ minRows: 2, maxRows: 5 }}
          />

          <ConnectionFieldTooltip title={t('databases.edit.serverCaCertificateTooltip')} />
        </div>
      </ConnectionFieldRow>
    </>
  );
};
