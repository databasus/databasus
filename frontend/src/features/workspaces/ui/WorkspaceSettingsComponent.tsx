import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { databaseApi } from '../../../entities/databases/api/databaseApi';
import type { User } from '../../../entities/users';
import { UserRole } from '../../../entities/users';
import { type Workspace, WorkspaceRole, workspaceApi } from '../../../entities/workspaces';
import { useIsMobile } from '../../../shared/hooks';
import { translateApiError } from '../../../shared/i18n';
import { WorkspaceAuditLogsComponent } from './WorkspaceAuditLogsComponent';
import { WorkspaceMembersComponent } from './WorkspaceMembersComponent';

interface Props {
  selectedWorkspace: Workspace;
  user: User;
  contentHeight: number;
}

export function WorkspaceSettingsComponent({ selectedWorkspace, user, contentHeight }: Props) {
  const { t } = useTranslation();
  const { message, modal } = App.useApp();
  const isMobile = useIsMobile();
  const [workspace, setWorkspace] = useState<Workspace | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const [formWorkspace, setFormWorkspace] = useState<Partial<Workspace>>({});
  const [nameError, setNameError] = useState(false);

  const [basicInfoChanges, setBasicInfoChanges] = useState(false);

  const isInstanceAdmin = user.role === UserRole.ADMIN;
  const canEdit =
    isInstanceAdmin ||
    selectedWorkspace.currentUserRole === WorkspaceRole.OWNER ||
    selectedWorkspace.currentUserRole === WorkspaceRole.ADMIN;
  const canDelete = isInstanceAdmin || selectedWorkspace.currentUserRole === WorkspaceRole.OWNER;

  useEffect(() => {
    loadWorkspace();
  }, [selectedWorkspace.id]);

  const checkBasicInfoChanges = (newFormWorkspace: Partial<Workspace>): boolean => {
    if (!workspace) return false;
    return newFormWorkspace.name !== workspace.name;
  };

  const loadWorkspace = async () => {
    setIsLoading(true);

    try {
      const workspaceData = await workspaceApi.getWorkspace(selectedWorkspace.id);
      setWorkspace(workspaceData);
      setFormWorkspace(workspaceData);
      setNameError(false);

      setBasicInfoChanges(false);
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsLoading(false);
    }
  };

  const handleFieldChange = <K extends keyof Workspace>(field: K, value: Workspace[K]) => {
    const newFormWorkspace = { ...formWorkspace, [field]: value };
    setFormWorkspace(newFormWorkspace);

    if (workspace) {
      setBasicInfoChanges(checkBasicInfoChanges(newFormWorkspace));
    }
  };

  const saveBasicInfo = async () => {
    if (!basicInfoChanges || !workspace || !canEdit) return;

    if (!formWorkspace.name?.trim()) {
      setNameError(true);
      message.error(t('workspaces.settings.nameRequired'));
      return;
    }
    setNameError(false);

    setIsSaving(true);
    try {
      const updatedWorkspace = await workspaceApi.updateWorkspace(
        workspace.id,
        formWorkspace.name.trim(),
      );
      setWorkspace(updatedWorkspace);
      setFormWorkspace(updatedWorkspace);

      setBasicInfoChanges(false);

      setNameError(false);
      message.success(t('workspaces.settings.updated'));
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsSaving(false);
    }
  };

  const handleDeleteWorkspace = async () => {
    if (!workspace) {
      message.error(t('workspaces.delete.notFound'));
      return;
    }

    if (!canDelete) {
      message.error(t('workspaces.delete.noPermission'));
      return;
    }

    modal.confirm({
      title: t('workspaces.delete.confirmation.title'),
      content: (
        <div>
          <p>
            <Trans
              i18nKey="workspaces.delete.confirmation.question"
              components={{ workspaceName: <strong>{workspace.name}</strong> }}
            />
          </p>
          <p className="mt-2 text-red-600">
            <Trans
              i18nKey="workspaces.delete.confirmation.warning"
              components={{ bold: <strong /> }}
            />
          </p>
        </div>
      ),
      okText: t('workspaces.delete.confirmation.submit'),
      okType: 'danger',
      cancelText: t('common.actions.cancel'),
      onOk: async () => {
        setIsDeleting(true);
        try {
          // Check if there are any databases in the workspace
          const databases = await databaseApi.listAllDatabases(workspace.id);

          if (databases.length > 0) {
            message.error(t('workspaces.delete.hasDatabases', { count: databases.length }));
            return;
          }

          await workspaceApi.deleteWorkspace(workspace.id);
          message.success(t('workspaces.delete.deleted'));
          window.location.href = '/';
        } catch (error: unknown) {
          message.error(translateApiError(error, t));
        } finally {
          setIsDeleting(false);
        }
      },
    });
  };

  return (
    <div className="flex grow">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className={`grow overflow-y-auto rounded bg-white shadow dark:bg-gray-800 ${isMobile ? 'p-3' : 'p-5'}`}
          style={{ height: contentHeight }}
        >
          <h1 className="mb-6 text-2xl font-bold dark:text-white">
            {t('workspaces.settings.title')}
          </h1>

          {isLoading || !workspace ? (
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          ) : (
            <>
              {!canEdit && (
                <div className="my-4 max-w-[500px] rounded-md bg-yellow-50 p-3 dark:bg-yellow-900/30">
                  <div className="text-sm text-yellow-800 dark:text-yellow-200">
                    {t('workspaces.settings.readOnly')}
                  </div>
                </div>
              )}

              <div className="space-y-6 text-sm">
                <div className="max-w-2xl border-b border-gray-200 pb-6 dark:border-gray-700">
                  <div className="max-w-md">
                    <div className="mb-1 font-medium text-gray-900 dark:text-white">
                      {t('workspaces.fields.name')}
                    </div>
                    <Input
                      value={formWorkspace.name || ''}
                      onChange={(e) => {
                        setNameError(false);
                        handleFieldChange('name', e.target.value);
                      }}
                      disabled={!canEdit}
                      placeholder={t('workspaces.fields.namePlaceholder')}
                      maxLength={100}
                      status={nameError ? 'error' : undefined}
                    />
                  </div>

                  {basicInfoChanges && canEdit && (
                    <div className="mt-4 flex space-x-2">
                      <Button
                        type="primary"
                        onClick={saveBasicInfo}
                        loading={isSaving}
                        disabled={isSaving}
                        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                      >
                        {isSaving
                          ? t('workspaces.settings.saving')
                          : t('workspaces.settings.saveChanges')}
                      </Button>

                      <Button
                        type="default"
                        onClick={() => {
                          if (workspace) {
                            const updatedForm = { ...formWorkspace, name: workspace.name };
                            setFormWorkspace(updatedForm);
                            setBasicInfoChanges(false);
                            setNameError(false);
                          }
                        }}
                        disabled={isSaving}
                      >
                        {t('workspaces.settings.reset')}
                      </Button>
                    </div>
                  )}
                </div>

                <div className="max-w-2xl border-b border-gray-200 pb-6 dark:border-gray-700">
                  <WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={user} />
                </div>

                {canDelete && (
                  <div className="max-w-2xl border-b border-gray-200 pb-6 dark:border-gray-700">
                    <h2 className="mb-4 text-xl font-bold text-gray-900 dark:text-white">
                      {t('workspaces.delete.dangerZone')}
                    </h2>

                    <div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/30">
                      <div
                        className={`flex ${isMobile ? 'flex-col gap-3' : 'items-start justify-between'}`}
                      >
                        <div className="flex-1">
                          <div className="font-medium text-red-900 dark:text-red-200">
                            {t('workspaces.delete.title')}
                          </div>
                          <div className="mt-1 text-sm text-red-700 dark:text-red-300">
                            {t('workspaces.delete.description')}
                          </div>
                        </div>

                        <div className={isMobile ? '' : 'ml-4'}>
                          <Button
                            type="primary"
                            danger
                            onClick={handleDeleteWorkspace}
                            disabled={!canDelete || isDeleting || isSaving}
                            loading={isDeleting}
                            className={`bg-red-600 hover:bg-red-700 ${isMobile ? 'w-full' : ''}`}
                          >
                            {isDeleting
                              ? t('workspaces.delete.deleting')
                              : t('workspaces.delete.submit')}
                          </Button>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                <WorkspaceAuditLogsComponent
                  workspaceId={workspace.id}
                  scrollContainerRef={scrollContainerRef}
                />
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
