import { useTranslation } from 'react-i18next';

import { type MongodbSettings } from '@/entities/databases';
import { ShowSshTunnelComponent } from '@/pages/databases/ui/show/ShowSshTunnelComponent';

interface Props {
  settings: MongodbSettings;
}

export const ShowMongoDbSpecificDataComponent = ({ settings }: Props) => {
  const { t } = useTranslation();

  return (
    <div>
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

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.cpuCount')}</div>
        <div>{settings.cpuCount}</div>
      </div>

      {settings.isDirectConnection && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.directConnection')}</div>
          <div>{t('common.answers.yes')}</div>
        </div>
      )}

      {settings.authDatabase && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.authDatabase')}</div>
          <div>{settings.authDatabase}</div>
        </div>
      )}

      {!!settings.excludeCollections?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeCollections')}</div>
          <div>{settings.excludeCollections.join(', ')}</div>
        </div>
      )}
    </div>
  );
};
