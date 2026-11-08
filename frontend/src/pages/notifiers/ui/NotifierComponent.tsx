import { ActionIcon, Alert, Button, Loader, TextInput } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { ArrowRight, CircleAlert, Info, Pencil, Trash2, X } from 'lucide-react';
import { useEffect, useEffectEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { notifierApi } from '@/entities/notifiers';
import type { Notifier } from '@/entities/notifiers';
import { EditNotifierComponent } from '@/features/edit-notifier';
import { NotifierTransferDialogComponent } from '@/pages/notifiers/ui/NotifierTransferDialogComponent';
import { ShowNotifierComponent } from '@/pages/notifiers/ui/show/ShowNotifierComponent';
import { useSessionMutation } from '@/shared/api';
import { ApiError, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { ConfirmationComponent, LoadFailure } from '@/shared/ui';

interface Props {
  onUnavailable?: () => void;
  notifierId: string;
  onNotifierChanged: (notifier: Notifier) => void;
  onNotifierDeleted: () => void;
  onNotifierTransferred: () => void;
  isCanManageNotifiers: boolean;
}

export const NotifierComponent = ({
  onUnavailable,
  notifierId,
  onNotifierChanged,
  onNotifierDeleted,
  onNotifierTransferred,
  isCanManageNotifiers,
}: Props) => {
  const testNotifierMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof notifierApi.testNotifier>) =>
      notifierApi.testNotifier(...parameters),
  });
  const deleteNotifierMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof notifierApi.deleteNotifier>) =>
      notifierApi.deleteNotifier(...parameters),
  });
  const updateNotifierMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof notifierApi.updateNotifier>) =>
      notifierApi.updateNotifier(...parameters),
  });

  const [actionError, setError] = useState<unknown>();
  const [deliveryError, setDeliveryError] = useState<string>();
  const client = useQueryClient();
  const query = useSessionQuery({
    queryKey: ['notifier', notifierId],
    queryFn: (signal) => notifierApi.getNotifier(notifierId, { signal }),
  });
  const [isUnavailable, setIsUnavailable] = useState(false);
  const isAccessRefused =
    query.error instanceof ApiError && [403, 404].includes(query.error.status ?? 0);
  const notifier = isAccessRefused || isUnavailable ? undefined : query.data;
  const error = actionError ?? query.error ?? undefined;
  const setNotifier = (resource: Notifier) => {
    void client.cancelQueries({ queryKey: query.queryKey, exact: true });
    client.setQueryData(query.queryKey, resource);
  };

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editName, setEditName] = useState('');
  const isSaving = updateNotifierMutation.isPending;

  const isSendingTestNotification = testNotifierMutation.isPending;

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const isRemoving = deleteNotifierMutation.isPending;

  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);

  const isCurrentNotifier = (revision: number) =>
    isCurrent() && notifierRevision.IsCurrent(revision);

  const sendTestNotification = async () => {
    if (!notifier || !isCurrent() || isSendingTestNotification) return;

    const revision = notifierRevision.GetCurrent();
    const testRevision = savedTestRevision.Invalidate();
    const isActive = () => isCurrentNotifier(revision) && savedTestRevision.IsCurrent(testRevision);

    setError(undefined);
    setDeliveryError(undefined);

    try {
      const testResult = await testNotifierMutation.mutateAsync([notifier.id]);
      if (!isActive()) return;

      const acknowledgedNotifier = latestNotifier.current;

      if (!acknowledgedNotifier) return;

      const testedNotifier = {
        ...acknowledgedNotifier,
        lastSendError: testResult.error ?? null,
      };

      if (testResult.isDelivered) {
        notify({
          kind: 'success',
          title: i18n.t('notifiers.testNotification.sent.title'),
          description: i18n.t('notifiers.testNotification.sent.description'),
          lifetimeMs: 4500,
          position: 'top-right',
        });
      } else {
        setDeliveryError(testResult.error ?? '');
      }

      if (testedNotifier.lastSendError !== acknowledgedNotifier.lastSendError) {
        setNotifier(testedNotifier);
        onNotifierChanged(testedNotifier);
      }
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    if (isActive()) testNotifierMutation.reset();
  };

  const remove = async () => {
    if (!notifier || !isCurrent()) return;

    const revision = notifierRevision.GetCurrent();

    try {
      await deleteNotifierMutation.mutateAsync([notifier.id]);
      if (!isCurrentNotifier(revision)) return;

      onNotifierDeleted();
    } catch (e) {
      if (!isCurrentNotifier(revision)) return;

      setError(e);
      setIsShowRemoveConfirm(false);
    }

    if (isCurrentNotifier(revision)) deleteNotifierMutation.reset();
  };

  const startEdit = (type: 'name' | 'settings') => {
    if (isEditName || isEditSettings || isSaving) return;

    if (type === 'settings') {
      savedTestRevision.Invalidate();
      testNotifierMutation.reset();
      setError(undefined);
      setDeliveryError(undefined);
    }

    setEditName(notifier?.name ?? '');
    setIsEditName(type === 'name');
    setIsEditSettings(type === 'settings');
  };

  // The settings of the response carry no secrets, so the backend keeps the stored ones.
  const saveName = async () => {
    if (!notifier || !isCurrent() || isSaving) return;

    const submittedName = editName;
    const revision = notifierRevision.GetCurrent();

    try {
      const savedNotifier = await updateNotifierMutation.mutateAsync([
        notifier.id,
        editName,
        notifier.settings,
      ]);
      if (!isCurrentNotifier(revision)) return;

      setNotifier(savedNotifier);
      const currentName = latestName.current;
      const isNewerName = currentName !== submittedName && currentName !== savedNotifier.name;

      if (!isNewerName) {
        setEditName(savedNotifier.name);
        setIsEditName(false);
      }

      onNotifierChanged(savedNotifier);
    } catch (e) {
      if (!isCurrentNotifier(revision)) return;

      setError(e);
    }

    if (isCurrentNotifier(revision)) updateNotifierMutation.reset();
  };

  const loadSettings = () => {
    if (!isCurrent()) return;

    notifierRevision.Invalidate();
    savedTestRevision.Invalidate();
    testNotifierMutation.reset();
    updateNotifierMutation.reset();
    deleteNotifierMutation.reset();
    setDeliveryError(undefined);
    setError(undefined);

    void query.refetch();
  };
  const reportUnavailable = useEffectEvent(() => {
    if (isAccessRefused) {
      setIsUnavailable(true);
      onUnavailable?.();
    }
  });
  useEffect(() => {
    if (query.error !== null) reportUnavailable();
  }, [query.error, query.errorUpdatedAt]);
  useEffect(() => {
    if (query.data !== undefined) setIsUnavailable(false);
  }, [query.data, query.dataRevision]);
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const notifierRevision = useRequestRevision();
  const savedTestRevision = useRequestRevision();
  const latestNotifier = useLatestRef(notifier);
  const latestName = useLatestRef(editName);

  useEffect(() => {
    setIsEditName(false);
    setIsEditSettings(false);
    setIsShowTransferDialog(false);
    setIsShowRemoveConfirm(false);
    notifierRevision.Invalidate();
    setError(undefined);

    return () => {
      notifierRevision.Invalidate();
    };
  }, [notifierId, notifierRevision]);

  const isNameUnsaved = isEditName && editName !== notifier?.name;

  return (
    <div className="flex min-h-0 w-full min-w-0 flex-col">
      <div className="min-h-0 min-w-0 grow rounded bg-surface p-4 text-content shadow sm:p-5">
        {error !== undefined &&
          (notifier ? (
            <Alert
              color="danger"
              icon={
                <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
              }
              title={translateApiError(error, t)}
            />
          ) : (
            <LoadFailure error={error} onRetry={loadSettings} />
          ))}
        {deliveryError !== undefined && (
          <Alert
            color="danger"
            title={t('notifiers.testNotification.notDelivered', { error: deliveryError })}
          />
        )}
        {!notifier && error === undefined ? (
          <div className="mt-10 flex justify-center">
            <Loader role="status" />
          </div>
        ) : notifier ? (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex min-w-0 items-center text-2xl font-bold break-words">
                <span className="min-w-0 wrap-anywhere">{notifier.name}</span>
                {isCanManageNotifiers && (
                  <ActionIcon
                    variant="subtle"
                    size="xs"
                    className="ml-2 shrink-0"
                    aria-label={t('common.actions.edit')}
                    onClick={() => startEdit('name')}
                    disabled={isEditSettings || isSaving}
                  >
                    <Pencil aria-hidden size="1em" className="inline-block shrink-0 align-middle" />
                  </ActionIcon>
                )}
              </div>
            ) : (
              <div>
                <div className="flex items-center">
                  <TextInput
                    aria-label={t('common.fields.name')}
                    className="max-w-62.5"
                    value={editName}
                    onChange={(e) => {
                      setEditName(e.target.value);
                    }}
                    placeholder={t('notifiers.config.namePlaceholder')}
                    size="md"
                  />

                  <div className="ml-1 flex items-center">
                    <Button
                      variant="subtle"
                      aria-label={t('common.actions.cancel')}
                      className="flex h-6 w-6 items-center justify-center p-0"
                      onClick={() => {
                        setIsEditName(false);
                      }}
                    >
                      <X
                        aria-hidden
                        className="inline-block shrink-0 align-middle text-muted"
                        size="1em"
                      />
                    </Button>
                  </div>
                </div>

                {isNameUnsaved && (
                  <Button
                    className="mt-1"
                    variant="filled"
                    onClick={() => saveName()}
                    aria-label={t('common.actions.save')}
                    loading={isSaving}
                    disabled={!editName}
                    aria-busy={isSaving}
                  >
                    {t('common.actions.save')}
                  </Button>
                )}
              </div>
            )}

            {notifier.lastSendError && (
              <div className="max-w-100 rounded border border-danger px-3 py-3">
                <div className="mt-1 flex items-center text-sm font-bold text-danger">
                  <Info className="mr-2 inline-block shrink-0 align-middle" size="1em" />
                  {t('notifiers.config.sendError.title')}
                </div>

                <div className="mt-3 text-sm">
                  {t('notifiers.config.sendError.errorLabel')}
                  <br />
                  {notifier.lastSendError}
                </div>

                <div className="mt-3 text-sm break-words whitespace-pre-wrap text-muted">
                  {t('notifiers.config.sendError.clearHint')}
                  <ul>
                    <li>{t('notifiers.config.sendError.clearBySendingTest')}</li>
                    <li>{t('notifiers.config.sendError.clearByNextNotification')}</li>
                  </ul>
                </div>
              </div>
            )}

            <div className="mt-5 flex items-center font-bold">
              <div>{t('notifiers.config.settingsTitle')}</div>

              {!isEditSettings && isCanManageNotifiers ? (
                <ActionIcon
                  variant="subtle"
                  size="xs"
                  className="ml-2"
                  aria-label={t('common.actions.edit')}
                  onClick={() => startEdit('settings')}
                  disabled={isEditName || isSaving}
                >
                  <Pencil aria-hidden size="1em" className="inline-block shrink-0 align-middle" />
                </ActionIcon>
              ) : (
                <div />
              )}
            </div>

            <div className="mt-1 text-sm">
              {isEditSettings ? (
                <EditNotifierComponent
                  workspaceId={notifier.workspaceId}
                  isShowClose
                  onClose={() => {
                    setIsEditSettings(false);
                    loadSettings();
                  }}
                  isShowName={false}
                  editingNotifier={notifier}
                  onPersisted={(savedNotifier) => {
                    if (!isCurrent() || savedNotifier.id !== notifierId) return;

                    savedTestRevision.Invalidate();
                    setNotifier(savedNotifier);
                    onNotifierChanged(savedNotifier);
                  }}
                  onChanged={() => setIsEditSettings(false)}
                />
              ) : (
                <ShowNotifierComponent notifier={notifier} />
              )}
            </div>

            {!isEditSettings && (
              <div className="mt-5 flex flex-wrap items-center gap-1">
                <Button
                  variant="filled"
                  aria-label={t('notifiers.testNotification.send')}
                  onClick={sendTestNotification}
                  loading={isSendingTestNotification}
                  disabled={isSendingTestNotification}
                  aria-busy={isSendingTestNotification}
                >
                  {t('notifiers.testNotification.send')}
                </Button>

                {isCanManageNotifiers && (
                  <>
                    <ActionIcon
                      variant="outline"
                      color="accent"
                      aria-label={t('notifiers.transfer.submit')}
                      onClick={() => setIsShowTransferDialog(true)}
                    >
                      <ArrowRight
                        aria-hidden
                        size="1em"
                        className="inline-block shrink-0 align-middle"
                      />
                    </ActionIcon>

                    <ActionIcon
                      variant="outline"
                      color="danger"
                      aria-label={t('common.actions.remove')}
                      onClick={() => setIsShowRemoveConfirm(true)}
                      loading={isRemoving}
                      disabled={isRemoving}
                      aria-busy={isRemoving}
                    >
                      <Trash2
                        aria-hidden
                        size="1em"
                        className="inline-block shrink-0 align-middle"
                      />
                    </ActionIcon>
                  </>
                )}
              </div>
            )}
          </div>
        ) : null}

        {isShowRemoveConfirm && (
          <ConfirmationComponent
            onConfirm={remove}
            onDecline={() => setIsShowRemoveConfirm(false)}
            description={t('notifiers.config.removeConfirmation')}
            actionText={t('common.actions.remove')}
            isDestructive
          />
        )}
      </div>

      {isShowTransferDialog && notifier && (
        <NotifierTransferDialogComponent
          notifier={notifier}
          onClose={() => setIsShowTransferDialog(false)}
          onTransferred={() => {
            setIsShowTransferDialog(false);
            onNotifierTransferred();
          }}
        />
      )}
    </div>
  );
};
