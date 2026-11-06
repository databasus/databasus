import { useTranslation } from 'react-i18next';

import { DATABASE_TYPE_LABEL_KEYS, type MysqlSettings, DatabaseType } from '@/entities/databases';
import { ShowSshTunnelComponent } from '@/pages/databases/ui/show/ShowSshTunnelComponent';

interface Props {
  settings: MysqlSettings;
}

export const ShowMySqlSpecificDataComponent = ({ settings }: Props) => {
  const { t } = useTranslation();

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">
          {t('databases.fields.version', {
            engine: t(DATABASE_TYPE_LABEL_KEYS[DatabaseType.MYSQL]),
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

      {!!settings.excludeTables?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTables')}</div>
          <div>{settings.excludeTables.join(', ')}</div>
        </div>
      )}
    </div>
  );
};
