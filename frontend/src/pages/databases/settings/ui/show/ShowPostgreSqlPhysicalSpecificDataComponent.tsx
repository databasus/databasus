import { useTranslation } from 'react-i18next';

import {
  type PostgresqlPhysicalSettings,
  PHYSICAL_DATABASE_BACKUP_TYPE_LABEL_KEYS,
  POSTGRESQL_SHORT_NAME,
  POSTGRES_SSL_MODE_LABEL_KEYS,
  PostgresSslMode,
} from '@/entities/databases';
import { ShowSshTunnelComponent } from '@/pages/databases/settings/ui/show/ShowSshTunnelComponent';

interface Props {
  settings: PostgresqlPhysicalSettings;
}

export const ShowPostgreSqlPhysicalSpecificDataComponent = ({ settings }: Props) => {
  const { t } = useTranslation();

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">
          {t('databases.fields.version', { engine: POSTGRESQL_SHORT_NAME })}
        </div>
        <div>{settings.version ?? ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.backupType')}</div>
        <div>
          {settings.backupType
            ? t(PHYSICAL_DATABASE_BACKUP_TYPE_LABEL_KEYS[settings.backupType])
            : ''}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2 break-all">{t('common.fields.host')}</div>
        <div>{settings.host || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.port')}</div>
        <div>{settings.port || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
        <div>{settings.username || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.password')}</div>
        <div>{'*************'}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.sslMode')}</div>
        <div>{t(POSTGRES_SSL_MODE_LABEL_KEYS[settings.sslMode ?? PostgresSslMode.Disable])}</div>
      </div>

      {!!settings.sslClientCert && settings.sslMode !== PostgresSslMode.Disable && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.clientCertificate')}</div>
          <div>*************</div>
        </div>
      )}

      <ShowSshTunnelComponent sshTunnel={settings.sshTunnel} />
    </div>
  );
};
