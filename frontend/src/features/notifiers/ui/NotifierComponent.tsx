import {
  ArrowRightOutlined,
  CloseOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Button, Input, Spin } from 'antd';
import { useState } from 'react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { notifierApi } from '../../../entities/notifiers';
import type { Notifier } from '../../../entities/notifiers';
import { translateApiError } from '../../../shared/i18n';
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
  const { t } = useTranslation();
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

  const sendTestNotification = async () => {
    if (!notifier) return;

    setIsSendingTestNotification(true);

    try {
      const testResult = await notifierApi.testNotifier(notifier.id);
      const testedNotifier = {
        ...notifier,
        lastSendError: testResult.error ?? null,
      };

      if (testResult.isDelivered) {
        ToastHelper.showToast({
          title: t('notifiers.testNotification.sent.title'),
          description: t('notifiers.testNotification.sent.description'),
        });
      } else {
        alert(
          t('notifiers.testNotification.notDelivered', {
            error: testResult.error,
          }),
        );
      }

      if (testedNotifier.lastSendError !== notifier.lastSendError) {
        setNotifier(testedNotifier);
        onNotifierChanged(testedNotifier);
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSendingTestNotification(false);
  };

  const remove = async () => {
    if (!notifier) return;

    setIsRemoving(true);

    try {
      await notifierApi.deleteNotifier(notifier.id);
      onNotifierDeleted();
    } catch (e) {
      alert(translateApiError(e, t));
      setIsShowRemoveConfirm(false);
    }

    setIsRemoving(false);
  };

  const startEdit = (type: 'name' | 'settings') => {
    setEditName(notifier?.name ?? '');
    setIsEditName(type === 'name');
    setIsEditSettings(type === 'settings');
    setIsNameUnsaved(false);
  };

  // The settings of the response carry no secrets, so the backend keeps the stored ones.
  const saveName = async () => {
    if (!notifier) return;

    setIsSaving(true);

    try {
      const savedNotifier = await notifierApi.updateNotifier(
        notifier.id,
        editName,
        notifier.settings,
      );
      setNotifier(savedNotifier);
      setIsNameUnsaved(false);
      setIsEditName(false);
      onNotifierChanged(savedNotifier);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  const loadSettings = () => {
    setNotifier(undefined);
    notifierApi
      .getNotifier(notifierId)
      .then(setNotifier)
      .catch((e: unknown) => alert(translateApiError(e, t)));
  };

  useEffect(() => {
    loadSettings();
  }, [notifierId]);

  return (
    <div className="w-full">
      <div className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800">
        {!notifier ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex items-center text-2xl font-bold">
                {notifier.name}
                {isCanManageNotifiers && (
                  <div className="ml-2 cursor-pointer" onClick={() => startEdit('name')}>
                    <img src="/icons/pen-gray.svg" />
                  </div>
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
                      className="flex h-6 w-6 items-center justify-center p-0"
                      onClick={() => {
                        setIsEditName(false);
                        setIsNameUnsaved(false);
                      }}
                    >
                      <CloseOutlined className="text-gray-500 dark:text-gray-400" />
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
              <div className="max-w-[400px] rounded border border-red-600 px-3 py-3">
                <div className="mt-1 flex items-center text-sm font-bold text-red-600">
                  <InfoCircleOutlined className="mr-2" style={{ color: 'red' }} />
                  {t('notifiers.config.sendError.title')}
                </div>

                <div className="mt-3 text-sm">
                  {t('notifiers.config.sendError.errorLabel')}
                  <br />
                  {notifier.lastSendError}
                </div>

                <div className="mt-3 text-sm break-words whitespace-pre-wrap text-gray-500 dark:text-gray-400">
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
                <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('settings')}>
                  <img src="/icons/pen-gray.svg" />
                </div>
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
                  onChanged={(savedNotifier) => {
                    setNotifier(savedNotifier);
                    setIsEditSettings(false);
                    onNotifierChanged(savedNotifier);
                  }}
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
                      icon={<ArrowRightOutlined />}
                      onClick={() => setIsShowTransferDialog(true)}
                      className="mr-1"
                    />

                    <Button
                      type="primary"
                      ghost
                      danger
                      icon={<DeleteOutlined />}
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
            actionButtonColor="red"
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
