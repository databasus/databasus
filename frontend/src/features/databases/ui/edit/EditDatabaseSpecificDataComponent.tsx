import { App, Modal } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, databaseApi } from '../../../../entities/databases';
import type { PostgresqlRestoreOptions } from '../../../../entities/restores';
import { translateApiError } from '../../../../shared/i18n';
import { CreateReadOnlyComponent } from './CreateReadOnlyComponent';
import { EditMariaDbSpecificDataComponent } from './EditMariaDbSpecificDataComponent';
import { EditMongoDbSpecificDataComponent } from './EditMongoDbSpecificDataComponent';
import { EditMySqlSpecificDataComponent } from './EditMySqlSpecificDataComponent';
import { EditPostgreSqlLogicalSpecificDataComponent } from './EditPostgreSqlLogicalSpecificDataComponent';
import { EditPostgreSqlPhysicalSpecificDataComponent } from './EditPostgreSqlPhysicalSpecificDataComponent';
import { toDatabaseSettingsTarget } from './toDatabaseSettingsTarget';

interface Props {
  database: Database;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  // Restore options come only from a PostgreSQL logical form in restore mode.
  onSaved: (database: Database, restoreOptions?: PostgresqlRestoreOptions) => void;

  isShowDbName?: boolean;
  isRestoreMode?: boolean;

  onConnectionErrorChange?: (hasConnectionError: boolean) => void;
}

export const EditDatabaseSpecificDataComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  isShowDbName = true,
  isRestoreMode = false,
  onConnectionErrorChange,
}: Props) => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const [isShowReadOnlyDialog, setIsShowReadOnlyDialog] = useState(false);
  const [editingDatabase, setEditingDatabase] = useState<Database>(database);

  const saveDb = async (databaseToSave: Database, restoreOptions?: PostgresqlRestoreOptions) => {
    setEditingDatabase(databaseToSave);

    if (!isSaveToApi) {
      onSaved(databaseToSave, restoreOptions);
      return;
    }

    try {
      const limitedUserCheck = await databaseApi.checkLimitedUser(
        toDatabaseSettingsTarget(databaseToSave, databaseToSave.settings),
      );

      if (limitedUserCheck.isAdvisable) {
        setIsShowReadOnlyDialog(true);
      } else {
        onSaved(databaseToSave);
      }
    } catch (e) {
      message.error(translateApiError(e, t));
    }
  };

  const onReadOnlyUserCreated = (updatedDatabase: Database) => {
    setEditingDatabase(updatedDatabase);
    setIsShowReadOnlyDialog(false);
  };

  const skipReadOnlyUser = () => {
    setIsShowReadOnlyDialog(false);
    onSaved(editingDatabase);
  };

  if (isShowReadOnlyDialog) {
    return (
      <Modal
        title={t('databases.readOnlyUser.dialogTitle')}
        footer={<div />}
        open={isShowReadOnlyDialog}
        onCancel={() => setIsShowReadOnlyDialog(false)}
        maskClosable={false}
        width={640}
      >
        <CreateReadOnlyComponent
          database={editingDatabase}
          onReadOnlyUserUpdated={onReadOnlyUserCreated}
          onGoBack={() => {
            setIsShowReadOnlyDialog(false);
          }}
          onSkipped={() => {
            skipReadOnlyUser();
          }}
          onReadOnlyUserNotSuggested={() => {
            onSaved(editingDatabase);
          }}
        />
      </Modal>
    );
  }

  const commonProps = {
    database: editingDatabase,
    isShowCancelButton,
    onCancel,
    isShowBackButton,
    onBack,
    saveButtonText,
    isSaveToApi,
    onSaved: saveDb,
    isShowDbName,
  };

  const { settings } = editingDatabase;

  switch (settings.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return (
        <EditPostgreSqlLogicalSpecificDataComponent
          {...commonProps}
          settings={settings}
          isRestoreMode={isRestoreMode}
        />
      );
    case DatabaseType.POSTGRES_PHYSICAL:
      return (
        <EditPostgreSqlPhysicalSpecificDataComponent
          {...commonProps}
          settings={settings}
          onConnectionErrorChange={onConnectionErrorChange}
        />
      );
    case DatabaseType.MYSQL:
      return <EditMySqlSpecificDataComponent {...commonProps} settings={settings} />;
    case DatabaseType.MARIADB:
      return <EditMariaDbSpecificDataComponent {...commonProps} settings={settings} />;
    case DatabaseType.MONGODB:
      return <EditMongoDbSpecificDataComponent {...commonProps} settings={settings} />;
  }
};
