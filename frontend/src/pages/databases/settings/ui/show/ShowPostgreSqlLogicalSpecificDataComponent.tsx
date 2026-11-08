import { useTranslation } from 'react-i18next';

import {
  type PostgresqlLogicalSettings,
  POSTGRESQL_SHORT_NAME,
  POSTGRES_SSL_MODE_LABEL_KEYS,
  PostgresSslMode,
} from '@/entities/databases';
import { ShowSshTunnelComponent } from '@/pages/databases/settings/ui/show/ShowSshTunnelComponent';

interface Props {
  settings: PostgresqlLogicalSettings;
}

export const ShowPostgreSqlLogicalSpecificDataComponent = ({ settings }: Props) => {
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
        <div className="min-w-[150px] pr-2">{t('databases.fields.databaseName')}</div>
        <div>{settings.database || ''}</div>
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

      {!!settings.includeSchemas?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.includeSchemas')}</div>
          <div>{settings.includeSchemas.join(', ')}</div>
        </div>
      )}

      {!!settings.excludeTables?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeWholeTables')}</div>
          <div>{settings.excludeTables.join(', ')}</div>
        </div>
      )}

      {!!settings.excludeTableData?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTableData')}</div>
          <div>{settings.excludeTableData.join(', ')}</div>
        </div>
      )}

      {!!settings.isSkipUserMappings && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.skipUserMappings')}</div>
          <div>{t('common.answers.yes')}</div>
        </div>
      )}
    </div>
  );
};
