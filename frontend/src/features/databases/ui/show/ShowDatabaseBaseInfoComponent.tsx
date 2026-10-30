import { useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  type Database,
  getDatabaseLogoFromType,
} from '../../../../entities/databases';

interface Props {
  database: Database;
  isShowName?: boolean;
  isShowType?: boolean;
}

export const ShowDatabaseBaseInfoComponent = ({ database, isShowName, isShowType }: Props) => {
  const { t } = useTranslation();

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.name')}</div>
          <div>{database.name || ''}</div>
        </div>
      )}

      {isShowType && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.databaseType')}</div>
          <div className="flex items-center">
            <span>{t(DATABASE_TYPE_LABEL_KEYS[database.settings.type])}</span>
            <img
              src={getDatabaseLogoFromType(database.settings.type)}
              alt=""
              className="ml-2 h-4 w-4"
            />
          </div>
        </div>
      )}
    </div>
  );
};
