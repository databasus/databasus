import { Button, Input, Select } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  NOTIFIER_TYPE_LABEL_KEYS,
  type Notifier,
  type NotifierSettings,
  NotifierType,
  getNotifierLogoFromType,
  notifierApi,
} from '../../../../entities/notifiers';
import { translateApiError } from '../../../../shared/i18n';
import { ToastHelper } from '../../../../shared/toast';
import { createDefaultNotifierSettings } from '../../lib/createDefaultNotifierSettings';
import { isNotifierSettingsFilled } from '../../lib/isNotifierSettingsFilled';
import { EditDiscordNotifierComponent } from './notifiers/EditDiscordNotifierComponent';
import { EditEmailNotifierComponent } from './notifiers/EditEmailNotifierComponent';
import { EditMattermostNotifierComponent } from './notifiers/EditMattermostNotifierComponent';
import { EditSlackNotifierComponent } from './notifiers/EditSlackNotifierComponent';
import { EditTeamsNotifierComponent } from './notifiers/EditTeamsNotifierComponent';
import { EditTelegramNotifierComponent } from './notifiers/EditTelegramNotifierComponent';
import { EditWebhookNotifierComponent } from './notifiers/EditWebhookNotifierComponent';

interface Props {
  workspaceId: string;

  isShowClose: boolean;
  onClose: () => void;

  isShowName: boolean;

  editingNotifier?: Notifier;
  onChanged: (notifier: Notifier) => void;
}

const SELECTABLE_NOTIFIER_TYPES: NotifierType[] = [
  NotifierType.TELEGRAM,
  NotifierType.EMAIL,
  NotifierType.WEBHOOK,
  NotifierType.SLACK,
  NotifierType.DISCORD,
  NotifierType.TEAMS,
  NotifierType.MATTERMOST,
];

export function EditNotifierComponent({
  workspaceId,
  isShowClose,
  onClose,
  isShowName,
  editingNotifier,
  onChanged,
}: Props) {
  const { t } = useTranslation();
  const [name, setName] = useState('');
  const [settings, setSettings] = useState<NotifierSettings | undefined>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isSendingTestNotification, setIsSendingTestNotification] = useState(false);
  const [isTestNotificationSuccess, setIsTestNotificationSuccess] = useState(false);

  const changeSettings = (changedSettings: NotifierSettings) => {
    setSettings(changedSettings);
    setIsUnsaved(true);
    setIsTestNotificationSuccess(false);
  };

  const save = async () => {
    if (!settings) return;

    setIsSaving(true);

    try {
      const savedNotifier = editingNotifier
        ? await notifierApi.updateNotifier(editingNotifier.id, name, settings)
        : await notifierApi.createNotifier(workspaceId, name, settings);
      onChanged(savedNotifier);
      setIsUnsaved(false);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  const sendTestNotification = async () => {
    if (!settings) return;

    setIsSendingTestNotification(true);

    try {
      const testResult = await notifierApi.testNotifierSettings(
        workspaceId,
        settings,
        editingNotifier?.id,
      );

      if (testResult.isDelivered) {
        setIsTestNotificationSuccess(true);
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

        if (settings.type === NotifierType.SLACK) {
          alert(t('notifiers.slack.testFailedHint'));
        }
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSendingTestNotification(false);
  };

  useEffect(() => {
    setIsUnsaved(false);
    setIsTestNotificationSuccess(false);
    setName(editingNotifier?.name ?? '');
    setSettings(
      editingNotifier
        ? structuredClone(editingNotifier.settings)
        : createDefaultNotifierSettings(NotifierType.TELEGRAM),
    );
  }, [editingNotifier]);

  if (!settings) return <div />;

  const isSaved = editingNotifier !== undefined;
  const isAllDataFilled = !!name && isNotifierSettingsFilled(settings, isSaved);

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('common.fields.name')}</div>

          <Input
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('notifiers.create.namePlaceholder')}
          />
        </div>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('common.fields.type')}</div>

        <div className="flex items-center">
          {isSaved ? (
            t(NOTIFIER_TYPE_LABEL_KEYS[settings.type])
          ) : (
            <Select
              value={settings.type}
              options={SELECTABLE_NOTIFIER_TYPES.map((type) => ({
                label: t(NOTIFIER_TYPE_LABEL_KEYS[type]),
                value: type,
              }))}
              onChange={(type: NotifierType) => changeSettings(createDefaultNotifierSettings(type))}
              size="small"
              className="w-[250px] max-w-[250px]"
            />
          )}

          <img src={getNotifierLogoFromType(settings.type)} className="ml-2 h-4 w-4" />
        </div>
      </div>

      <div className="mt-5" />

      <div>
        {settings.type === NotifierType.TELEGRAM && (
          <EditTelegramNotifierComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.EMAIL && (
          <EditEmailNotifierComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.WEBHOOK && (
          <EditWebhookNotifierComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.SLACK && (
          <EditSlackNotifierComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.DISCORD && (
          <EditDiscordNotifierComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.TEAMS && (
          <EditTeamsNotifierComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.MATTERMOST && (
          <EditMattermostNotifierComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}
      </div>

      <div className="mt-3 flex">
        {isUnsaved && !isTestNotificationSuccess ? (
          <Button
            className="mr-1"
            disabled={isSendingTestNotification || !isAllDataFilled}
            loading={isSendingTestNotification}
            type="primary"
            onClick={sendTestNotification}
          >
            {t('notifiers.testNotification.send')}
          </Button>
        ) : (
          <div />
        )}

        {isUnsaved && isTestNotificationSuccess ? (
          <Button
            className="mr-1"
            disabled={isSaving || !isAllDataFilled}
            loading={isSaving}
            type="primary"
            onClick={save}
          >
            {t('common.actions.save')}
          </Button>
        ) : (
          <div />
        )}

        {isShowClose ? (
          <Button
            className="mr-1"
            disabled={isSaving}
            type="primary"
            danger
            ghost
            onClick={onClose}
          >
            {t('common.actions.cancel')}
          </Button>
        ) : (
          <div />
        )}
      </div>
    </div>
  );
}
