import { useTranslation } from 'react-i18next';

import { DATABASE_TYPE_LABEL_KEYS, type MariadbSettings, DatabaseType } from '@/entities/databases';
import { ShowSshTunnelComponent } from '@/pages/databases/ui/show/ShowSshTunnelComponent';

interface Props {
  settings: MariadbSettings;
}

export const ShowMariaDbSpecificDataComponent = ({ settings }: Props) => {
  const { t } = useTranslation();

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">
          {t('databases.fields.version', {
            engine: t(DATABASE_TYPE_LABEL_KEYS[DatabaseType.MARIADB]),
          })}
        </div>
        <div>{settings.version || ''}</div>
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
        <div className="min-w-[150px] pr-2">{t('databases.fields.useHttps')}</div>
        <div>{settings.isHttps ? t('common.answers.yes') : t('common.answers.no')}</div>
      </div>
      <ShowSshTunnelComponent sshTunnel={settings.sshTunnel} />

      {settings.isExcludeEvents && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeEvents')}</div>
          <div>{t('common.answers.yes')}</div>
        </div>
      )}

      {settings.isSkipGaleraDisable && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.galeraReplication')}</div>
          <div>{t('databases.show.galeraDisableSkipped')}</div>
        </div>
      )}

      {!!settings.excludeTables?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTables')}</div>
          <div>{settings.excludeTables.join(', ')}</div>
        </div>
      )}
    </div>
  );
};
