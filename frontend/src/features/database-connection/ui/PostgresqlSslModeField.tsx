import { Select } from 'antd';
import { useTranslation } from 'react-i18next';

import { POSTGRES_SSL_MODE_LABEL_KEYS, PostgresSslMode } from '@/entities/databases';

import { ConnectionFieldRow } from './ConnectionFieldRow';

interface Props {
  controlId: string;
  sslMode?: PostgresSslMode;
  onChange: (sslMode: PostgresSslMode) => void;
}

export const PostgresqlSslModeField = ({ controlId, sslMode, onChange }: Props) => {
  const { t } = useTranslation();

  return (
    <ConnectionFieldRow label={t('databases.fields.sslMode')} controlId={`${controlId}-ssl-mode`}>
      <Select
        id={`${controlId}-ssl-mode`}
        value={sslMode ?? PostgresSslMode.Disable}
        onChange={onChange}
        options={Object.values(PostgresSslMode).map((mode) => ({
          label: t(POSTGRES_SSL_MODE_LABEL_KEYS[mode]),
          value: mode,
        }))}
        size="small"
        className="w-full max-w-[200px]"
      />
    </ConnectionFieldRow>
  );
};
