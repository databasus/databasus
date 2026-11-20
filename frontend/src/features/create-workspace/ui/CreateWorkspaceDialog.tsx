import { Button } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { type InstanceSettings, type User, UserRole } from '@/entities/users';
import type { Workspace } from '@/entities/workspaces';
import { CreateWorkspaceForm } from '@/features/create-workspace/ui/CreateWorkspaceForm';
import { Dialog } from '@/shared/ui';

interface Props {
  user: User;
  globalSettings: InstanceSettings;
  workspacesCount: number;
  onClose: () => void;
  onWorkspaceCreated: (workspace: Workspace) => void;
}

export const CreateWorkspaceDialog = ({
  user,
  globalSettings,
  workspacesCount,
  onClose,
  onWorkspaceCreated,
}: Props) => {
  const { t } = useTranslation();
  const isAllowedToCreateWorkspaces =
    globalSettings.isMemberAllowedToCreateWorkspaces || user.role === UserRole.ADMIN;

  if (!isAllowedToCreateWorkspaces) {
    return (
      <Dialog title={t('workspaces.create.permissionDenied.title')} opened onClose={onClose}>
        <p>{t('workspaces.create.permissionDenied.description')}</p>
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <Button variant="filled" onClick={onClose}>
            {t('common.actions.ok')}
          </Button>
        </div>
      </Dialog>
    );
  }

  return (
    <Dialog title={t('workspaces.create.title')} opened onClose={onClose}>
      <CreateWorkspaceForm
        workspacesCount={workspacesCount}
        onCancel={onClose}
        onWorkspaceCreated={(workspace) => {
          onWorkspaceCreated(workspace);
          onClose();
        }}
      />
    </Dialog>
  );
};
