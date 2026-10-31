import {
  ArrowRightOutlined,
  CloseOutlined,
  DeleteOutlined,
  EditOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Button, Input, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { notifierApi } from '../../../entities/notifiers';
import type { Notifier } from '../../../entities/notifiers';
import { useSessionGuard } from '../../../shared/auth';
import { translateApiError } from '../../../shared/i18n';
import { RequestRevision } from '../../../shared/lib/RequestRevision';
import { ToastHelper } from '../../../shared/toast';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditNotifierComponent } from './edit/EditNotifierComponent';
import { NotifierTransferDialogComponent } from './NotifierTransferDialogComponent';
import { ShowNotifierComponent } from './show/ShowNotifierComponent';

interface Props {
  notifierId: string;
  onNotifierChanged: (notifier: Notifier) => void;
  onNotifierDeleted: () => void;
  onNotifierTransferred: () => void;
  isCanManageNotifiers: boolean;
}

export const NotifierComponent = ({
  notifierId,
  onNotifierChanged,
  onNotifierDeleted,
  onNotifierTransferred,
  isCanManageNotifiers,
}: Props) => {
  const [notifier, setNotifier] = useState<Notifier | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editName, setEditName] = useState('');
  const [isNameUnsaved, setIsNameUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isSendingTestNotification, setIsSendingTestNotification] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);

  const isCurrentNotifier = (revision: number) =>
    isCurrent() && notifierRevision.current.IsCurrent(revision);

  const sendTestNotification = async () => {
    if (!notifier || !isCurrent()) return;
    const revision = notifierRevision.current.GetCurrent();

    setIsSendingTestNotification(true);

    try {
      const testResult = await notifierApi.testNotifier(notifier.id);
      if (!isCurrentNotifier(revision)) return;
      const testedNotifier = {
        ...notifier,
        lastSendError: testResult.error ?? null,
      };

      if (testResult.isDelivered) {
        ToastHelper.showToast({
          title: i18n.t('notifiers.testNotification.sent.title'),
          description: i18n.t('notifiers.testNotification.sent.description'),
        });
      } else {
        alert(
          i18n.t('notifiers.testNotification.notDelivered', {
            error: testResult.error,
          }),
        );
      }

      if (testedNotifier.lastSendError !== notifier.lastSendError) {
        setNotifier(testedNotifier);
        onNotifierChanged(testedNotifier);
      }
    } catch (e) {
      if (!isCurrentNotifier(revision)) return;
      alert(translateApiError(e, i18n.t));
    }

    if (isCurrentNotifier(revision)) setIsSendingTestNotification(false);
  };

  const remove = async () => {
    if (!notifier || !isCurrent()) return;
    const revision = notifierRevision.current.GetCurrent();

    setIsRemoving(true);

    try {
      await notifierApi.deleteNotifier(notifier.id);
      if (!isCurrentNotifier(revision)) return;
      onNotifierDeleted();
    } catch (e) {
      if (!isCurrentNotifier(revision)) return;
      alert(translateApiError(e, i18n.t));
      setIsShowRemoveConfirm(false);
    }

    if (isCurrentNotifier(revision)) setIsRemoving(false);
  };

  const startEdit = (type: 'name' | 'settings') => {
    setEditName(notifier?.name ?? '');
    setIsEditName(type === 'name');
    setIsEditSettings(type === 'settings');
    setIsNameUnsaved(false);
  };

  // The settings of the response carry no secrets, so the backend keeps the stored ones.
  const saveName = async () => {
    if (!notifier || !isCurrent()) return;
    const revision = notifierRevision.current.GetCurrent();

    setIsSaving(true);

    try {
      const savedNotifier = await notifierApi.updateNotifier(
        notifier.id,
        editName,
        notifier.settings,
      );
      if (!isCurrentNotifier(revision)) return;
      setNotifier(savedNotifier);
      setIsNameUnsaved(false);
      setIsEditName(false);
      onNotifierChanged(savedNotifier);
    } catch (e) {
      if (!isCurrentNotifier(revision)) return;
      alert(translateApiError(e, i18n.t));
    }

    if (isCurrentNotifier(revision)) setIsSaving(false);
  };

  const loadSettings = () => {
    if (!isCurrent()) return;
    const revision = notifierRevision.current.Invalidate();
    setNotifier(undefined);
    notifierApi
      .getNotifier(notifierId)
      .then((loadedNotifier) => {
        if (isCurrentNotifier(revision)) setNotifier(loadedNotifier);
      })
      .catch((e: unknown) => {
        if (isCurrentNotifier(revision)) alert(translateApiError(e, i18n.t));
      });
  };

  const { t, i18n } = useTranslation();
  const isCurrent = useSessionGuard();
  const notifierRevision = useRef(new RequestRevision());

  useEffect(() => {
    loadSettings();
    return () => {
      notifierRevision.current.Invalidate();
    };
  }, [notifierId]);

  return (
    <div className="flex min-h-0 w-full min-w-0 flex-col">
      <div className="min-h-0 min-w-0 grow overflow-y-auto rounded bg-surface p-4 text-content shadow sm:p-5">
        {!notifier ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : (
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
                  />
                )}
              </div>
            ) : (
              <div>
                <div className="flex items-center">
                  <Input
                    className="max-w-[250px]"
                    value={editName}
                    onChange={(e) => {
                      setEditName(e.target.value);
                      setIsNameUnsaved(true);
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
                        setIsNameUnsaved(false);
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
                    loading={isSaving}
                    disabled={!editName}
                  >
                    {t('common.actions.save')}
                  </Button>
                )}
              </div>
            )}

            {notifier.lastSendError && (
              <div className="max-w-[400px] rounded border border-danger px-3 py-3">
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
        )}

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
