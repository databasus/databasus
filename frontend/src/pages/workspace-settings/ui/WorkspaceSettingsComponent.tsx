import { Alert, Button, Loader, TextInput } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { CircleAlert } from 'lucide-react';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { databaseApi } from '@/entities/databases';
import type { User } from '@/entities/users';
import { UserRole } from '@/entities/users';
import { type Workspace, WorkspaceRole, workspaceApi } from '@/entities/workspaces';
import { WorkspaceAuditLogsComponent } from '@/pages/workspace-settings/ui/WorkspaceAuditLogsComponent';
import { WorkspaceMembersComponent } from '@/pages/workspace-settings/ui/WorkspaceMembersComponent';
import { useSessionMutation } from '@/shared/api';
import { ApiError, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { useIsMobile } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { Dialog } from '@/shared/ui';
import { LoadFailure } from '@/shared/ui';

interface Props {
  selectedWorkspace: Workspace;
  user: User;
  onRenamed?: (workspace: Workspace) => void;
  onDeleted?: (workspaceId: string) => void;
  onUnavailable?: (workspaceId: string) => void;
}

export function WorkspaceSettingsComponent({
  selectedWorkspace,
  user,
  onRenamed,
  onDeleted,
  onUnavailable,
}: Props) {
  const client = useQueryClient();
  const updateWorkspaceMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof workspaceApi.updateWorkspace>) =>
      workspaceApi.updateWorkspace(...parameters),
  });
  const deleteWorkspaceMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof workspaceApi.deleteWorkspace>) =>
      workspaceApi.deleteWorkspace(...parameters),
  });

  const callbacks = useLatestRef({ onRenamed, onDeleted, onUnavailable });
  const isCurrent = useSessionGuard();
  const [workspace, setWorkspace] = useState<Workspace | undefined>(undefined);
  const query = useSessionQuery({
    queryKey: ['workspace-settings', selectedWorkspace.id],
    queryFn: (signal) => workspaceApi.getWorkspace(selectedWorkspace.id, { signal }),
  });
  const deletionCheck = useSessionQuery({
    queryKey: ['workspace-deletion-check', selectedWorkspace.id],
    enabled: false,
    queryFn: (signal) => databaseApi.listAllDatabases(selectedWorkspace.id, { signal }),
  });
  const isLoading = query.isPending;
  const [refusalError, setLoadError] = useState<unknown>();
  const loadError = refusalError ?? query.error ?? undefined;
  const [mutationError, setMutationError] = useState<unknown>();
  const isSaving = updateWorkspaceMutation.isPending;
  const [isDeleting, setDeleting] = useState(false);
  const deletionAttempt = useRef<{ isWriteStarted: boolean } | undefined>(undefined);

  const [formWorkspace, setFormWorkspace] = useState<Partial<Workspace>>({});
  const [isNameError, setNameError] = useState(false);

  const isActiveWorkspace = (revision: number) =>
    isCurrent() && workspaceRevision.IsCurrent(revision);

  const handleUnavailable = (error: unknown) => {
    if (!isCurrent()) return;

    workspaceRevision.Invalidate();
    deletionAttempt.current = undefined;
    void client.cancelQueries({ queryKey: deletionCheck.queryKey, exact: true });
    setDeletionTarget(undefined);
    setDeleting(false);
    setWorkspace(undefined);
    setFormWorkspace({});
    setLoadError(error);
    updateWorkspaceMutation.reset();
    deleteWorkspaceMutation.reset();
    callbacks.current.onUnavailable?.(selectedWorkspace.id);
  };

  const handleMutationFailure = (error: unknown) => {
    if (error instanceof ApiError && error.status === 404) {
      handleUnavailable(error);
      return;
    }

    setMutationError(error);
  };

  const loadWorkspace = () => {
    setLoadError(undefined);
    return query.refetch();
  };
  useEffect(() => {
    if (!query.data || !isCurrent()) return;

    setWorkspace(query.data);
    setFormWorkspace(query.data);
    setNameError(false);
  }, [query.data, query.dataRevision, isCurrent]);
  const reportRefusal = useEffectEvent(() =>
    callbacks.current.onUnavailable?.(selectedWorkspace.id),
  );
  useEffect(() => {
    if (query.error instanceof ApiError && [403, 404].includes(query.error.status ?? 0))
      reportRefusal();
  }, [query.error]);

  const saveBasicInfo = async () => {
    if (isSaving || !workspace || !canEdit || formWorkspace.name === workspace.name) return;

    const submittedName = formWorkspace.name ?? '';

    if (!submittedName.trim()) {
      setNameError(true);
      return;
    }

    const revision = workspaceRevision.GetCurrent();
    setNameError(false);
    setMutationError(undefined);

    try {
      const updatedWorkspace = await updateWorkspaceMutation.mutateAsync([
        workspace.id,
        submittedName.trim(),
      ]);
      if (!isActiveWorkspace(revision)) return;

      setWorkspace(updatedWorkspace);
      setFormWorkspace((current) => (current.name === submittedName ? updatedWorkspace : current));
      callbacks.current.onRenamed?.(updatedWorkspace);
      notify({ kind: 'success', title: i18n.t('workspaces.settings.updated') });
    } catch (error) {
      if (isActiveWorkspace(revision)) handleMutationFailure(error);
    } finally {
      if (isActiveWorkspace(revision)) updateWorkspaceMutation.reset();
    }
  };

  const handleDeleteWorkspace = async () => {
    if (!workspace) {
      notify({ kind: 'error', title: t('workspaces.delete.notFound') });
      return;
    }

    if (!canDelete) {
      notify({ kind: 'error', title: t('workspaces.delete.noPermission') });
      return;
    }

    setDeletionTarget({ workspace, revision: workspaceRevision.GetCurrent() });
  };

  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const isMobile = useIsMobile();

  const workspaceRevision = useRequestRevision();
  const nameControlId = useId();
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const [deletionTarget, setDeletionTarget] = useState<{
    workspace: Workspace;
    revision: number;
  }>();
  const isInstanceAdmin = user.role === UserRole.ADMIN;
  const canEdit =
    isInstanceAdmin ||
    selectedWorkspace.currentUserRole === WorkspaceRole.OWNER ||
    selectedWorkspace.currentUserRole === WorkspaceRole.ADMIN;
  const canDelete = isInstanceAdmin || selectedWorkspace.currentUserRole === WorkspaceRole.OWNER;

  useEffect(
    () => () => {
      workspaceRevision.Invalidate();
      setDeletionTarget(undefined);
    },
    [workspaceRevision],
  );

  const cancelDeletion = () => {
    if (deletionAttempt.current?.isWriteStarted) return;

    deletionAttempt.current = undefined;
    void client.cancelQueries({ queryKey: deletionCheck.queryKey, exact: true });
    setDeleting(false);
    setDeletionTarget(undefined);
  };

  const deleteWorkspace = async () => {
    if (!deletionTarget || deletionAttempt.current) return;

    const { workspace, revision } = deletionTarget;
    if (!isActiveWorkspace(revision)) return;

    const attempt = { isWriteStarted: false };
    deletionAttempt.current = attempt;
    const isActiveDeletion = () =>
      deletionAttempt.current === attempt && isActiveWorkspace(revision);
    setDeleting(true);

    try {
      const { data: databases, error } = await deletionCheck.refetch();
      if (error) throw error;
      if (!databases) return;
      if (!isActiveDeletion()) return;

      if (databases.length > 0) {
        notify({
          kind: 'error',
          title: t('workspaces.delete.hasDatabases', { count: databases.length }),
        });
        return;
      }

      attempt.isWriteStarted = true;
      await deleteWorkspaceMutation.mutateAsync([workspace.id]);
      if (!isActiveDeletion()) return;

      notify({ kind: 'success', title: t('workspaces.delete.deleted') });
      callbacks.current.onDeleted?.(workspace.id);
    } catch (error: unknown) {
      if (isActiveDeletion()) handleMutationFailure(error);
    } finally {
      if (deletionAttempt.current === attempt) {
        deletionAttempt.current = undefined;

        if (isActiveWorkspace(revision)) {
          deleteWorkspaceMutation.reset();
          setDeleting(false);
          setDeletionTarget((current) => (current === deletionTarget ? undefined : current));
        }
      }
    }
  };

  const isBasicInfoChanged = !!workspace && formWorkspace.name !== workspace.name;

  return (
    <div className="flex min-h-0 min-w-0 grow">
      <div className="flex min-h-0 w-full min-w-0 flex-col">
        <div
          ref={scrollContainerRef}
          className={`min-h-0 grow overflow-y-auto rounded bg-surface shadow ${isMobile ? 'p-3' : 'p-5'}`}
        >
          <h1 className="mb-6 text-2xl font-bold">{t('workspaces.settings.title')}</h1>

          {isLoading ? (
            <Loader size="md" role="status" />
          ) : loadError !== undefined ? (
            <LoadFailure error={loadError} onRetry={() => void loadWorkspace()} />
          ) : workspace ? (
            <>
              {mutationError !== undefined && (
                <Alert
                  color="danger"
                  icon={
                    <CircleAlert
                      aria-hidden
                      size={16}
                      className="inline-block shrink-0 align-middle"
                    />
                  }
                  title={translateApiError(mutationError, t)}
                />
              )}
              {!canEdit && (
                <div className="my-4 max-w-125 rounded-md bg-surface p-3">
                  <div className="text-sm text-warning">{t('workspaces.settings.readOnly')}</div>
                </div>
              )}

              <div className="space-y-6 text-sm">
                <div className="max-w-2xl border-b border-border pb-6">
                  <div className="max-w-md">
                    <label htmlFor={nameControlId} className="mb-1 block font-medium text-content">
                      {t('workspaces.fields.name')}
                    </label>
                    <TextInput
                      id={nameControlId}
                      value={formWorkspace.name || ''}
                      onChange={(e) => {
                        setNameError(false);
                        setFormWorkspace((current) => ({ ...current, name: e.target.value }));
                      }}
                      disabled={!canEdit}
                      placeholder={t('workspaces.fields.namePlaceholder')}
                      maxLength={100}
                      error={isNameError ? t('workspaces.settings.nameRequired') : undefined}
                      errorProps={{ role: 'alert', className: 'text-sm' }}
                      classNames={{ wrapper: 'mb-0' }}
                    />
                  </div>

                  {isBasicInfoChanged && canEdit && (
                    <div className="mt-4 flex flex-wrap gap-2">
                      <Button
                        variant="filled"
                        onClick={saveBasicInfo}
                        loading={isSaving}
                        disabled={isSaving}
                        aria-busy={isSaving}
                      >
                        {isSaving
                          ? t('workspaces.settings.saving')
                          : t('workspaces.settings.saveChanges')}
                      </Button>

                      <Button
                        variant="default"
                        onClick={() => {
                          if (workspace) {
                            const updatedForm = { ...formWorkspace, name: workspace.name };
                            setFormWorkspace(updatedForm);
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

                <div className="max-w-2xl border-b border-border pb-6">
                  <WorkspaceMembersComponent
                    selectedWorkspace={selectedWorkspace}
                    user={user}
                    onUnavailable={handleUnavailable}
                  />
                </div>

                {canDelete && (
                  <div className="max-w-2xl border-b border-border pb-6">
                    <h2 className="mb-4 text-xl font-bold text-muted">
                      {t('workspaces.delete.dangerZone')}
                    </h2>

                    <div className="rounded-lg border border-border bg-surface p-4">
                      <div
                        className={`flex ${isMobile ? 'flex-col gap-3' : 'items-start justify-between'}`}
                      >
                        <div className="flex-1">
                          <div className="font-medium text-danger">
                            {t('workspaces.delete.title')}
                          </div>
                          <div className="mt-1 text-sm text-danger">
                            {t('workspaces.delete.description')}
                          </div>
                        </div>

                        <div className={isMobile ? '' : 'ml-4'}>
                          <Button
                            variant="filled"
                            color="danger"
                            onClick={handleDeleteWorkspace}
                            disabled={!canDelete || isDeleting || isSaving}
                            loading={isDeleting}
                            className={` ${isMobile ? 'w-full' : ''}`}
                            aria-busy={isDeleting}
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
                  onUnavailable={handleUnavailable}
                  scrollContainerRef={scrollContainerRef}
                />
              </div>
            </>
          ) : null}
        </div>
      </div>

      {deletionTarget && (
        <Dialog
          opened
          title={
            <span className="flex items-start gap-2">
              <CircleAlert aria-hidden size="1.25em" className="shrink-0 text-warning" />
              <span>{t('workspaces.delete.confirmation.title')}</span>
            </span>
          }
          onClose={cancelDeletion}
          closeOnEscape={!deleteWorkspaceMutation.isPending}
          closeOnClickOutside={!deleteWorkspaceMutation.isPending}
          withCloseButton={!deleteWorkspaceMutation.isPending}
        >
          <div>
            <p>
              <Trans
                i18nKey="workspaces.delete.confirmation.question"
                components={{ workspaceName: <strong>{deletionTarget.workspace.name}</strong> }}
              />
            </p>
            <p className="mt-2 text-danger">
              <Trans
                i18nKey="workspaces.delete.confirmation.warning"
                components={{ bold: <strong /> }}
              />
            </p>
          </div>

          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <Button onClick={cancelDeletion} disabled={deleteWorkspaceMutation.isPending}>
              {t('common.actions.cancel')}
            </Button>
            <Button
              color="danger"
              variant="outline"
              loading={isDeleting}
              disabled={isDeleting}
              onClick={() => void deleteWorkspace()}
              aria-busy={isDeleting}
            >
              {t('workspaces.delete.confirmation.submit')}
            </Button>
          </div>
        </Dialog>
      )}
    </div>
  );
}
