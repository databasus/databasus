import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Modal } from 'antd';
import { Spin } from 'antd';
import { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type User, UserRole, type InstanceSettings } from '../../../entities/users';
import { type Workspace, workspaceApi } from '../../../entities/workspaces';
import { translateApiError } from '../../../shared/i18n';

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
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [isCreating, setIsCreating] = useState(false);
  const [workspaceName, setWorkspaceName] = useState(
    workspacesCount === 0 ? t('workspaces.create.defaultName') : '',
  );

  const isAllowedToCreateWorkspaces =
    globalSettings.isMemberAllowedToCreateWorkspaces || user.role === UserRole.ADMIN;

  const handleCreateWorkspace = async () => {
    if (!workspaceName.trim()) {
      message.error(t('workspaces.create.nameRequired'));
      return;
    }

    setIsCreating(true);

    try {
      const newWorkspace = await workspaceApi.createWorkspace(workspaceName.trim());

      message.success(t('workspaces.create.created'));
      onWorkspaceCreated(newWorkspace);
      onClose();
    } catch (error) {
      message.error(translateApiError(error, t));
    } finally {
      setIsCreating(false);
    }
  };

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
          className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
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
        <div className="dark:text-gray-300">
          <Trans i18nKey="workspaces.create.description" components={{ lineBreak: <br /> }} />
        </div>

        <label className="mt-5 mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {t('workspaces.fields.name')}
        </label>
        <Input
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
