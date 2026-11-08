import { useTranslation } from 'react-i18next';

import { POSTGRES_SSL_MODE_LABEL_KEYS, PostgresSslMode } from '@/entities/databases';
import { ConnectionFieldRow } from '@/features/database-connection/ui/ConnectionFieldRow';
import { WrappingSelect } from '@/shared/ui';

interface Props {
  controlId: string;
  sslMode?: PostgresSslMode;
  onChange: (sslMode: PostgresSslMode) => void;
}

export const PostgresqlSslModeField = ({ controlId, sslMode, onChange }: Props) => {
  const { t } = useTranslation();

  return (
    <ConnectionFieldRow label={t('databases.fields.sslMode')} controlId={`${controlId}-ssl-mode`}>
      <WrappingSelect
        id={`${controlId}-ssl-mode`}
        value={sslMode ?? PostgresSslMode.Disable}
        onChange={onChange}
        data={Object.values(PostgresSslMode).map((mode) => ({
          label: t(POSTGRES_SSL_MODE_LABEL_KEYS[mode]),
          value: mode,
        }))}
        size="xs"
        className="w-full max-w-[200px]"
      />
    </ConnectionFieldRow>
  );
};
