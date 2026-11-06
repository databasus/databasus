import { Alert, Button, Input, Select, notification } from 'antd';
import { useEffect, useId } from 'react';
import { useTranslation } from 'react-i18next';

import {
  NOTIFIER_TYPE_LABEL_KEYS,
  type Notifier,
  NotifierType,
  getNotifierLogoFromType,
} from '@/entities/notifiers';
import { createDefaultNotifierSettings } from '@/features/edit-notifier/lib/createDefaultNotifierSettings';
import { isNotifierAddressChanged } from '@/features/edit-notifier/lib/isNotifierAddressChanged';
import { isNotifierSettingsFilled } from '@/features/edit-notifier/lib/isNotifierSettingsFilled';
import { useNotifierDraft } from '@/features/edit-notifier/model/useNotifierDraft';
import { EditDiscordNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditDiscordNotifierComponent';
import { EditEmailNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditEmailNotifierComponent';
import { EditMattermostNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditMattermostNotifierComponent';
import { EditSlackNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditSlackNotifierComponent';
import { EditTeamsNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditTeamsNotifierComponent';
import { EditTelegramNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditTelegramNotifierComponent';
import { EditWebhookNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditWebhookNotifierComponent';
import { translateApiError } from '@/shared/i18n';
import { WRAPPING_SELECT_CLASS_NAME } from '@/shared/ui';

interface Props {
  workspaceId: string;

  isShowClose: boolean;
  onClose: () => void;

  isShowName: boolean;

  editingNotifier?: Notifier;
  onChanged: (notifier: Notifier) => void;
  onPersisted: (notifier: Notifier) => void;
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
  onPersisted,
}: Props) {
  const { t } = useTranslation();
  const fieldId = useId();
  const noticeId = useId();
  const [notices, noticeHolder] = notification.useNotification();
  const form = useNotifierDraft({
    workspaceId,
    editingNotifier,
    onChanged,
    onPersisted,
    onTestSucceeded: () =>
      notices.success({
        key: noticeId,
        message: t('notifiers.testNotification.sent.title'),
        description: t('notifiers.testNotification.sent.description'),
        duration: 4,
        placement: 'bottomRight',
      }),
  });

  useEffect(
    () => () => notices.destroy(noticeId),
    [notices, noticeId, workspaceId, editingNotifier?.id],
  );

  const { name, settings, savedSettings, isUnsaved, isSaving, changeSettings, save } = form;
  const isSendingTestNotification = form.isTesting;
  const isTestNotificationSuccess = form.isTested;
  const sendTestNotification = form.testSettings;
  const isSaved = form.savedNotifierId !== undefined;
  const isAddressChanged = isNotifierAddressChanged(settings, savedSettings);
  const isAllDataFilled =
    !!name.trim() && isNotifierSettingsFilled(settings, isSaved, savedSettings);
  const errorText =
    form.error?.kind === 'api'
      ? translateApiError(form.error.error, t)
      : form.error?.kind === 'delivery'
        ? t('notifiers.testNotification.notDelivered', { error: form.error.diagnostic })
        : undefined;

  return (
    <section
      key={`${workspaceId}:${editingNotifier?.id ?? 'new'}`}
      aria-label={t('notifiers.config.settingsTitle')}
      className="min-h-0 min-w-0 text-content"
    >
      {noticeHolder}
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label htmlFor={fieldId} className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('common.fields.name')}
          </label>
          <Input
            id={fieldId}
            value={name}
            onChange={(event) => form.changeName(event.target.value)}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('notifiers.create.namePlaceholder')}
          />
        </div>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <label
          htmlFor={isSaved ? undefined : `${fieldId}-type`}
          className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2"
        >
          {t('common.fields.type')}
        </label>
        <div className="flex w-full min-w-0 items-center">
          {isSaved ? (
            <span>{t(NOTIFIER_TYPE_LABEL_KEYS[settings.type])}</span>
          ) : (
            <Select
              virtual={false}
              id={`${fieldId}-type`}
              value={settings.type}
              disabled={isSaving}
              options={SELECTABLE_NOTIFIER_TYPES.map((type) => ({
                label: t(NOTIFIER_TYPE_LABEL_KEYS[type]),
                value: type,
              }))}
              onChange={(type: NotifierType) => changeSettings(createDefaultNotifierSettings(type))}
              size="small"
              className={`w-full max-w-[250px] min-w-0 ${WRAPPING_SELECT_CLASS_NAME}`}
            />
          )}
          <img
            src={getNotifierLogoFromType(settings.type)}
            alt=""
            className="ml-2 h-4 w-4 shrink-0"
          />
        </div>
      </div>

      <div className="mt-5" />

      <div>
        {settings.type === NotifierType.TELEGRAM && (
          <EditTelegramNotifierComponent
            settings={settings}
            isSaved={isSaved}
            isAddressChanged={isAddressChanged}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.EMAIL && (
          <EditEmailNotifierComponent
            settings={settings}
            isSaved={isSaved}
            isAddressChanged={isAddressChanged}
            onChange={changeSettings}
          />
        )}

        {settings.type === NotifierType.WEBHOOK && (
          <EditWebhookNotifierComponent
            settings={settings}
            isSaved={isSaved}
            isAddressChanged={isAddressChanged}
            savedHeaders={savedSettings?.type === NotifierType.WEBHOOK ? savedSettings.headers : []}
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
            isAddressChanged={isAddressChanged}
            onChange={changeSettings}
          />
        )}
      </div>

      {errorText && (
        <Alert
          role="alert"
          type="error"
          showIcon
          message={
            <div className="max-h-40 overflow-auto font-mono whitespace-pre-wrap">{errorText}</div>
          }
          description={
            form.error?.kind === 'delivery' && form.error.isSlack
              ? t('notifiers.slack.testFailedHint')
              : undefined
          }
          className="mt-3 break-words"
        />
      )}

      <div className="mt-3 flex flex-wrap gap-2">
        {isUnsaved && !isTestNotificationSuccess ? (
          <Button
            className="!h-auto min-h-8 max-w-full text-left !whitespace-normal"
            disabled={isSendingTestNotification || isSaving || !isAllDataFilled}
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
            className="!h-auto min-h-8 max-w-full text-left !whitespace-normal"
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
            className="!h-auto min-h-8 max-w-full text-left !whitespace-normal"
            disabled={isSaving}
            onClick={onClose}
          >
            {t('common.actions.cancel')}
          </Button>
        ) : (
          <div />
        )}
      </div>
    </section>
  );
}
