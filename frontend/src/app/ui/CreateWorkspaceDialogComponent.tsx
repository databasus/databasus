import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Modal } from 'antd';
import { Spin } from 'antd';
import { useId, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type User, UserRole, type InstanceSettings } from '@/entities/users';
import { type Workspace, workspaceApi } from '@/entities/workspaces';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';

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
  const { t, i18n } = useTranslation();
  const [isCreating, setIsCreating] = useState(false);
  const [workspaceName, setWorkspaceName] = useState(
    workspacesCount === 0 ? t('workspaces.create.defaultName') : '',
  );

  const handleCreateWorkspace = async () => {
    if (isCreating) return;
    if (!workspaceName.trim()) {
      message.error(t('workspaces.create.nameRequired'));
      return;
    }

    setIsCreating(true);

    try {
      const newWorkspace = await workspaceApi.createWorkspace(workspaceName.trim());

      if (!isCurrent()) return;

      message.success(i18n.t('workspaces.create.created'));
      onWorkspaceCreated(newWorkspace);
      onClose();
    } catch (error) {
      if (isCurrent()) message.error(translateApiError(error, i18n.t));
    } finally {
      if (isCurrent()) setIsCreating(false);
    }
  };
  const { message } = App.useApp();

  const isAllowedToCreateWorkspaces =
    globalSettings.isMemberAllowedToCreateWorkspaces || user.role === UserRole.ADMIN;

  const isCurrent = useSessionGuard();
  const nameControlId = useId();

  if (!isAllowedToCreateWorkspaces) {
    return (
      <Modal
        title={t('workspaces.create.permissionDenied.title')}
        open
        onCancel={onClose}
        footer={[
          <Button key="ok" type="primary" onClick={onClose}>
            {t('common.actions.ok')}
          </Button>,
        ]}
      >
        <p>{t('workspaces.create.permissionDenied.description')}</p>
      </Modal>
    );
  }

  return (
    <Modal
      title={t('workspaces.create.title')}
      open
      onCancel={onClose}
      footer={[
        <Button key="cancel" onClick={onClose} disabled={isCreating}>
          {t('common.actions.cancel')}
        </Button>,

        <Button
          key="create"
          type="primary"
          onClick={handleCreateWorkspace}
          disabled={isCreating || !workspaceName.trim()}
        >
          {isCreating ? (
            <Spin indicator={<LoadingOutlined spin />} size="small" />
          ) : (
            t('workspaces.create.submit')
          )}
        </Button>,
      ]}
    >
      <div className="mb-4">
        <div className="">
          <Trans i18nKey="workspaces.create.description" components={{ lineBreak: <br /> }} />
        </div>

        <label htmlFor={nameControlId} className="mt-5 mb-2 block text-sm font-medium text-muted">
          {t('workspaces.fields.name')}
        </label>
        <Input
          id={nameControlId}
          value={workspaceName}
          onChange={(e) => setWorkspaceName(e.target.value)}
          placeholder={t('workspaces.fields.namePlaceholder')}
          disabled={isCreating}
          onPressEnter={handleCreateWorkspace}
          autoFocus
        />
      </div>
    </Modal>
  );
};
