import { Button, Loader, TextInput } from '@mantine/core';
import { useId, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type User, UserRole, type InstanceSettings } from '@/entities/users';
import { type Workspace, workspaceApi } from '@/entities/workspaces';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { Dialog } from '@/shared/ui';

interface Props {
  user: User;
  globalSettings: InstanceSettings;

  onClose: () => void;
  onWorkspaceCreated: (workspace: Workspace) => void;

  workspacesCount: number;
}

export const CreateWorkspaceDialogComponent = ({
  user,
  globalSettings,
  onClose,
  onWorkspaceCreated,
  workspacesCount,
}: Props) => {
  const createWorkspaceMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof workspaceApi.createWorkspace>) =>
      workspaceApi.createWorkspace(...parameters),
  });

  const { t, i18n } = useTranslation();
  const isCreating = createWorkspaceMutation.isPending;
  const [workspaceName, setWorkspaceName] = useState(
    workspacesCount === 0 ? t('workspaces.create.defaultName') : '',
  );

  const handleCreateWorkspace = async () => {
    if (isCreating) return;
    if (!workspaceName.trim()) {
      notify({ kind: 'error', title: t('workspaces.create.nameRequired') });
      return;
    }

    try {
      const newWorkspace = await createWorkspaceMutation.mutateAsync([workspaceName.trim()]);

      if (!isCurrent()) return;

      notify({ kind: 'success', title: i18n.t('workspaces.create.created') });
      onWorkspaceCreated(newWorkspace);
      onClose();
    } catch (error) {
      if (isCurrent()) notify({ kind: 'error', title: translateApiError(error, i18n.t) });
    } finally {
      if (isCurrent()) createWorkspaceMutation.reset();
    }
  };
  const notify = useFeedback();

  const isAllowedToCreateWorkspaces =
    globalSettings.isMemberAllowedToCreateWorkspaces || user.role === UserRole.ADMIN;

  const isCurrent = useSessionGuard();
  const nameControlId = useId();

  if (!isAllowedToCreateWorkspaces) {
    return (
      <Dialog title={t('workspaces.create.permissionDenied.title')} opened onClose={onClose}>
        <p>{t('workspaces.create.permissionDenied.description')}</p>
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          {[
            <Button key="ok" variant="filled" onClick={onClose}>
              {t('common.actions.ok')}
            </Button>,
          ]}
        </div>
      </Dialog>
    );
  }

  return (
    <Dialog title={t('workspaces.create.title')} opened onClose={onClose}>
      <div className="mb-4">
        <div className="">
          <Trans i18nKey="workspaces.create.description" components={{ lineBreak: <br /> }} />
        </div>

        <label htmlFor={nameControlId} className="mt-5 mb-2 block text-sm font-medium text-muted">
          {t('workspaces.fields.name')}
        </label>
        <TextInput
          id={nameControlId}
          value={workspaceName}
          onChange={(e) => setWorkspaceName(e.target.value)}
          placeholder={t('workspaces.fields.namePlaceholder')}
          disabled={isCreating}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              handleCreateWorkspace();
            }
          }}
          data-autofocus
        />
      </div>
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        {[
          <Button key="cancel" onClick={onClose} disabled={isCreating}>
            {t('common.actions.cancel')}
          </Button>,
          <Button
            key="create"
            variant="filled"
            onClick={handleCreateWorkspace}
            disabled={isCreating || !workspaceName.trim()}
          >
            {isCreating ? <Loader size="xs" role="status" /> : t('workspaces.create.submit')}
          </Button>,
        ]}
      </div>
    </Dialog>
  );
};
