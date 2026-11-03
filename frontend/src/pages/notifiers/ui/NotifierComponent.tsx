import {
  ArrowRightOutlined,
  CloseOutlined,
  DeleteOutlined,
  EditOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Alert, App, Button, Input, Spin } from 'antd';
import { useEffect, useEffectEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { EditNotifierComponent } from '@/features/edit-notifier';
import { ApiError } from '@/shared/api';

import { notifierApi } from '../../../entities/notifiers';
import type { Notifier } from '../../../entities/notifiers';
import { useSessionGuard } from '../../../shared/auth';
import { useLatestRef, useRequestRevision } from '../../../shared/hooks';
import { translateApiError } from '../../../shared/i18n';
import { ConfirmationComponent, LoadFailure } from '../../../shared/ui';
import { NotifierTransferDialogComponent } from './NotifierTransferDialogComponent';
import { ShowNotifierComponent } from './show/ShowNotifierComponent';

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
  const [error, setError] = useState<unknown>();
  const [deliveryError, setDeliveryError] = useState<string>();
  const [notifier, setNotifier] = useState<Notifier | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editName, setEditName] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const [isSendingTestNotification, setIsSendingTestNotification] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

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
    setIsSendingTestNotification(true);

    try {
      const testResult = await notifierApi.testNotifier(notifier.id);
      if (!isActive()) return;

      const acknowledgedNotifier = latestNotifier.current;

      if (!acknowledgedNotifier) return;

      const testedNotifier = {
        ...acknowledgedNotifier,
        lastSendError: testResult.error ?? null,
      };

      if (testResult.isDelivered) {
        notification.success({
          message: i18n.t('notifiers.testNotification.sent.title'),
          description: i18n.t('notifiers.testNotification.sent.description'),
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

    if (isActive()) setIsSendingTestNotification(false);
  };

  const remove = async () => {
    if (!notifier || !isCurrent()) return;

    const revision = notifierRevision.GetCurrent();

    setIsRemoving(true);

    try {
      await notifierApi.deleteNotifier(notifier.id);
      if (!isCurrentNotifier(revision)) return;

      onNotifierDeleted();
    } catch (e) {
      if (!isCurrentNotifier(revision)) return;

      setError(e);
      setIsShowRemoveConfirm(false);
    }

    if (isCurrentNotifier(revision)) setIsRemoving(false);
  };

  const startEdit = (type: 'name' | 'settings') => {
    if (isEditName || isEditSettings || isSaving) return;

    if (type === 'settings') {
      savedTestRevision.Invalidate();
      setIsSendingTestNotification(false);
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

    setIsSaving(true);

    try {
      const savedNotifier = await notifierApi.updateNotifier(
        notifier.id,
        editName,
        notifier.settings,
      );
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

    if (isCurrentNotifier(revision)) setIsSaving(false);
  };

  const loadSettings = () => {
    if (!isCurrent()) return;

    const revision = notifierRevision.Invalidate();
    savedTestRevision.Invalidate();
    setIsSendingTestNotification(false);
    setIsSaving(false);
    setIsRemoving(false);
    setDeliveryError(undefined);
    setError(undefined);
    setNotifier(undefined);

    notifierApi
      .getNotifier(notifierId)
      .then((loadedNotifier) => {
        if (isCurrentNotifier(revision)) setNotifier(loadedNotifier);
      })
      .catch((e: unknown) => {
        if (isCurrentNotifier(revision)) {
          setError(e);

          if (e instanceof ApiError && [403, 404].includes(e.status ?? 0)) onUnavailable?.();
        }
      });
  };

  const { t, i18n } = useTranslation();
  const { notification } = App.useApp();
  const isCurrent = useSessionGuard();
  const notifierRevision = useRequestRevision();
  const savedTestRevision = useRequestRevision();
  const latestNotifier = useLatestRef(notifier);
  const latestName = useLatestRef(editName);

  const startSettingsLoad = useEffectEvent(() => {
    void loadSettings();
  });

  useEffect(() => {
    setIsEditName(false);
    setIsEditSettings(false);
    setIsShowTransferDialog(false);
    setIsShowRemoveConfirm(false);
    startSettingsLoad();

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
            <Alert type="error" showIcon message={translateApiError(error, t)} />
          ) : (
            <LoadFailure error={error} onRetry={loadSettings} />
          ))}
        {deliveryError !== undefined && (
          <Alert
            type="error"
            message={t('notifiers.testNotification.notDelivered', { error: deliveryError })}
          />
        )}
        {!notifier && error === undefined ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : notifier ? (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex min-w-0 items-center text-2xl font-bold break-words">
                {notifier.name}
                {isCanManageNotifiers && (
                  <Button
                    type="text"
                    size="small"
                    className="ml-2 shrink-0"
                    aria-label={t('common.actions.edit')}
                    icon={<EditOutlined aria-hidden />}
                    onClick={() => startEdit('name')}
                    disabled={isEditSettings || isSaving}
                  />
                )}
              </div>
            ) : (
              <div>
                <div className="flex items-center">
                  <Input
                    aria-label={t('common.fields.name')}
                    className="max-w-62.5"
                    value={editName}
                    onChange={(e) => {
                      setEditName(e.target.value);
                    }}
                    placeholder={t('notifiers.config.namePlaceholder')}
                    size="large"
                  />

                  <div className="ml-1 flex items-center">
                    <Button
                      type="text"
                      aria-label={t('common.actions.cancel')}
                      className="flex h-6 w-6 items-center justify-center p-0"
                      onClick={() => {
                        setIsEditName(false);
                      }}
                    >
                      <CloseOutlined aria-hidden className="text-muted" />
                    </Button>
                  </div>
                </div>

                {isNameUnsaved && (
                  <Button
                    className="mt-1"
                    type="primary"
                    onClick={() => saveName()}
                    aria-label={t('common.actions.save')}
                    loading={isSaving}
                    disabled={!editName}
                  >
                    {t('common.actions.save')}
                  </Button>
                )}
              </div>
            )}

            {notifier.lastSendError && (
              <div className="max-w-100 rounded border border-danger px-3 py-3">
                <div className="mt-1 flex items-center text-sm font-bold text-danger">
                  <InfoCircleOutlined className="mr-2" />
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
                <Button
                  type="text"
                  size="small"
                  className="ml-2"
                  aria-label={t('common.actions.edit')}
                  icon={<EditOutlined aria-hidden />}
                  onClick={() => startEdit('settings')}
                  disabled={isEditName || isSaving}
                />
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
              <div className="mt-5">
                <Button
                  type="primary"
                  className="mr-1"
                  aria-label={t('notifiers.testNotification.send')}
                  onClick={sendTestNotification}
                  loading={isSendingTestNotification}
                  disabled={isSendingTestNotification}
                >
                  {t('notifiers.testNotification.send')}
                </Button>

                {isCanManageNotifiers && (
                  <>
                    <Button
                      type="primary"
                      ghost
                      aria-label={t('notifiers.transfer.submit')}
                      icon={<ArrowRightOutlined aria-hidden />}
                      onClick={() => setIsShowTransferDialog(true)}
                      className="mr-1"
                    />

                    <Button
                      type="primary"
                      ghost
                      danger
                      aria-label={t('common.actions.remove')}
                      icon={<DeleteOutlined aria-hidden />}
                      onClick={() => setIsShowRemoveConfirm(true)}
                      loading={isRemoving}
                      disabled={isRemoving}
                    />
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
