import { App, Button, Modal } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { DatabaseConnectionForm } from '@/features/database-connection';

import {
  type Database,
  type DatabaseEditDraft,
  databaseApi,
  toDatabaseSettingsTarget,
} from '../../../entities/databases';
import { useSessionGuard } from '../../../shared/auth';
import { translateApiError } from '../../../shared/i18n';
import { CreateLimitedUserComponent } from './CreateLimitedUserComponent';

interface Props {
  database: Database;
  onDatabaseSaved: (database: Database) => void;
  onClose: () => void;
}

const createEditDraft = (database: Database): DatabaseEditDraft => ({
  scenario: 'edit',
  workspaceId: database.workspaceId,
  databaseId: database.id,
  name: database.name,
  notifierIds: database.notifierIds,
  savedSettings: database.settings,
  settings: database.settings,
  hasUserChosenSslMode: true,
});

export const EditDatabaseConnectionSection = ({ database, onDatabaseSaved, onClose }: Props) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const isActive = useSessionGuard();
  const [draft, setDraft] = useState(() => createEditDraft(database));
  const [isShowLimitedUserDialog, setIsShowLimitedUserDialog] = useState(false);

  const offerLimitedUserOrClose = async (savedDraft: DatabaseEditDraft) => {
    try {
      const limitedUserCheck = await databaseApi.checkLimitedUser(
        toDatabaseSettingsTarget(savedDraft),
      );
      if (!isActive()) return;

      if (limitedUserCheck.isAdvisable) {
        setDraft(savedDraft);
        setIsShowLimitedUserDialog(true);
      } else {
        onClose();
      }
    } catch (e) {
      if (!isActive()) return;

      message.error(translateApiError(e, t));
      onClose();
    }
  };

  // The save is reported at once, because it stands whatever the limited user check answers. The
  // draft is rebased on it only together with opening the dialog, which unmounts the form, so a
  // replaced draft can never inherit a test approval.
  const saveDraft = async (submittedDraft: DatabaseEditDraft) => {
    const savedDatabase = await databaseApi.updateDatabase(
      submittedDraft.databaseId,
      submittedDraft,
    );
    if (!isActive()) return;

    onDatabaseSaved(savedDatabase);

    await offerLimitedUserOrClose(createEditDraft(savedDatabase));
  };

  if (isShowLimitedUserDialog) {
    return (
      <Modal
        title={t('databases.readOnlyUser.dialogTitle')}
        footer={<div />}
        open
        onCancel={() => setIsShowLimitedUserDialog(false)}
        maskClosable={false}
        width={640}
      >
        <CreateLimitedUserComponent
          draft={draft}
          onLimitedUserCreated={(limitedUser) => {
            setDraft({
              ...draft,
              settings: {
                ...draft.settings,
                username: limitedUser.username,
                password: limitedUser.password,
              },
            });
            setIsShowLimitedUserDialog(false);
          }}
          onGoBack={() => setIsShowLimitedUserDialog(false)}
          onSkipped={onClose}
          onLimitedUserNotSuggested={onClose}
        />
      </Modal>
    );
  }

  return (
    <DatabaseConnectionForm
      draft={draft}
      onDraftChange={setDraft}
      onSubmit={saveDraft}
      submitLabel={t('common.actions.save')}
      secondaryActions={<Button onClick={onClose}>{t('common.actions.cancel')}</Button>}
    />
  );
};
