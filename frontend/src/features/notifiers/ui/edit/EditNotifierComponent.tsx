import { Button, Input, Select } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  MattermostDeliveryMode,
  NOTIFIER_TYPE_LABEL_KEYS,
  type Notifier,
  NotifierType,
  WebhookMethod,
  getDefaultEmailNotifierSecurity,
  notifierApi,
  validateDiscordNotifier,
  validateEmailNotifier,
  validateMattermostNotifier,
  validateSlackNotifier,
  validateTeamsNotifier,
  validateTelegramNotifier,
  validateWebhookNotifier,
  validateSESNotifier,
  validateSNSNotifier,
} from '../../../../entity/notifiers';
import { getNotifierLogoFromType } from '../../../../entity/notifiers/models/getNotifierLogoFromType';
import { translateApiError } from '../../../../shared/i18n';
import { ToastHelper } from '../../../../shared/toast';
import { EditDiscordNotifierComponent } from './notifiers/EditDiscordNotifierComponent';
import { EditEmailNotifierComponent } from './notifiers/EditEmailNotifierComponent';
import { EditMattermostNotifierComponent } from './notifiers/EditMattermostNotifierComponent';
import { EditSESNotifierComponent } from './notifiers/EditSESNotifierComponent';
import { EditSlackNotifierComponent } from './notifiers/EditSlackNotifierComponent';
import { EditSNSNotifierComponent } from './notifiers/EditSNSNotifierComponent';
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

const SELECTABLE_NOTIFIER_TYPES = [
  NotifierType.TELEGRAM,
  NotifierType.EMAIL,
  NotifierType.WEBHOOK,
  NotifierType.SLACK,
  NotifierType.DISCORD,
  NotifierType.TEAMS,
  NotifierType.MATTERMOST,
  NotifierType.SES,
  NotifierType.SNS,
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
  const [notifier, setNotifier] = useState<Notifier | undefined>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isSendingTestNotification, setIsSendingTestNotification] = useState(false);
  const [isTestNotificationSuccess, setIsTestNotificationSuccess] = useState(false);

  const save = async () => {
    if (!notifier) return;

    setIsSaving(true);

    try {
      await notifierApi.saveNotifier(notifier);
      onChanged(notifier);
      setIsUnsaved(false);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  const sendTestNotification = async () => {
    if (!notifier) return;

    setIsSendingTestNotification(true);

    try {
      await notifierApi.sendTestNotificationDirect(notifier);
      setIsTestNotificationSuccess(true);
      ToastHelper.showToast({
        title: t('notifiers.testNotification.sent.title'),
        description: t('notifiers.testNotification.sent.description'),
      });
    } catch (e) {
      alert(translateApiError(e, t));

      if (notifier.notifierType === NotifierType.SLACK) {
        alert(t('notifiers.slack.testFailedHint'));
      }
    }

    setIsSendingTestNotification(false);
  };

  const setNotifierType = (type: NotifierType) => {
    if (!notifier) return;

    notifier.emailNotifier = undefined;
    notifier.telegramNotifier = undefined;
    notifier.teamsNotifier = undefined;
    notifier.webhookNotifier = undefined;
    notifier.slackNotifier = undefined;
    notifier.discordNotifier = undefined;
    notifier.mattermostNotifier = undefined;

    if (type === NotifierType.TELEGRAM) {
      notifier.telegramNotifier = {
        botToken: '',
        targetChatId: '',
      };
    }

    if (type === NotifierType.EMAIL) {
      const defaultSmtpPort = 0;

      notifier.emailNotifier = {
        targetEmail: '',
        smtpHost: '',
        smtpPort: defaultSmtpPort,
        smtpUser: '',
        smtpPassword: '',
        from: '',
        isInsecureSkipVerify: false,
        security: getDefaultEmailNotifierSecurity(defaultSmtpPort),
        heloName: '',
      };
    }

    if (type === NotifierType.WEBHOOK) {
      notifier.webhookNotifier = {
        webhookUrl: '',
        webhookMethod: WebhookMethod.POST,
        headers: [],
      };
    }

    if (type === NotifierType.SLACK) {
      notifier.slackNotifier = {
        botToken: '',
        targetChatId: '',
      };
    }

    if (type === NotifierType.DISCORD) {
      notifier.discordNotifier = {
        channelWebhookUrl: '',
      };
    }

    if (type === NotifierType.TEAMS) {
      notifier.teamsNotifier = { powerAutomateUrl: '' };
    }

    if (type === NotifierType.MATTERMOST) {
      notifier.mattermostNotifier = {
        deliveryMode: MattermostDeliveryMode.WEBHOOK,
        webhookUrl: '',
        serverUrl: '',
        botToken: '',
        targetChannelName: '',
        targetChannelId: '',
        overrideUsername: '',
        overrideIconUrl: '',
        isInsecureSkipVerify: false,
      };
    }

    if (type === NotifierType.SES) {
      notifier.sesNotifier = {
        fromEmail: '',
        targetEmail: '',
        region: 'us-east-1',
        authType: 'IAM_ROLE',
        accessKey: '',
        secretKey: '',
        roleArn: '',
        externalId: '',
      };
    }

    if (type === NotifierType.SNS) {
      notifier.snsNotifier = {
        topicArn: '',
        region: 'us-east-1',
        authType: 'KEYS',
        accessKey: '',
        secretKey: '',
        roleArn: '',
        externalId: '',
      };
    }

    setNotifier(
      JSON.parse(
        JSON.stringify({
          ...notifier,
          workspaceId,
          notifierType: type,
        }),
      ),
    );
  };

  useEffect(() => {
    setIsUnsaved(false);
    setNotifier(
      editingNotifier
        ? JSON.parse(JSON.stringify(editingNotifier))
        : {
            id: undefined as unknown as string,
            name: '',
            workspaceId,
            notifierType: NotifierType.TELEGRAM,
            telegramNotifier: {
              botToken: '',
              targetChatId: '',
            },
          },
    );
  }, [editingNotifier]);

  useEffect(() => {
    setIsTestNotificationSuccess(false);
  }, [notifier]);

  const isAllDataFilled = () => {
    if (!notifier) return false;
    if (!notifier.name) return false;

    if (notifier.notifierType === NotifierType.TELEGRAM && notifier.telegramNotifier) {
      return validateTelegramNotifier(!notifier.id, notifier.telegramNotifier);
    }

    if (notifier.notifierType === NotifierType.EMAIL && notifier.emailNotifier) {
      return validateEmailNotifier(notifier.emailNotifier);
    }

    if (notifier.notifierType === NotifierType.WEBHOOK && notifier.webhookNotifier) {
      return validateWebhookNotifier(!notifier.id, notifier.webhookNotifier);
    }

    if (notifier.notifierType === NotifierType.SLACK && notifier.slackNotifier) {
      return validateSlackNotifier(!notifier.id, notifier.slackNotifier);
    }

    if (notifier.notifierType === NotifierType.DISCORD && notifier.discordNotifier) {
      return validateDiscordNotifier(!notifier.id, notifier.discordNotifier);
    }

    if (notifier.notifierType === NotifierType.TEAMS && notifier.teamsNotifier) {
      return validateTeamsNotifier(!notifier.id, notifier.teamsNotifier);
    }

    if (notifier.notifierType === NotifierType.MATTERMOST && notifier.mattermostNotifier) {
      return validateMattermostNotifier(!notifier.id, notifier.mattermostNotifier);
    }

    if (notifier.notifierType === NotifierType.SES && notifier.sesNotifier) {
      return validateSESNotifier(notifier.sesNotifier);
    }

    if (notifier.notifierType === NotifierType.SNS && notifier.snsNotifier) {
      return validateSNSNotifier(notifier.snsNotifier);
    }

    return false;
  };

  if (!notifier) return <div />;

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('common.fields.name')}</div>

          <Input
            value={notifier?.name || ''}
            onChange={(e) => {
              setNotifier({ ...notifier, name: e.target.value });
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
          <Select
            value={notifier?.notifierType}
            options={SELECTABLE_NOTIFIER_TYPES.map((type) => ({
              label: t(NOTIFIER_TYPE_LABEL_KEYS[type]),
              value: type,
            }))}
            onChange={(value) => {
              setNotifierType(value);
              setIsUnsaved(true);
            }}
            size="small"
            className="w-[250px] max-w-[250px]"
          />

          <img src={getNotifierLogoFromType(notifier?.notifierType)} className="ml-2 h-4 w-4" />
        </div>
      </div>

      <div className="mt-5" />

      <div>
        {notifier?.notifierType === NotifierType.TELEGRAM && (
          <EditTelegramNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.EMAIL && (
          <EditEmailNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.WEBHOOK && (
          <EditWebhookNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.SLACK && (
          <EditSlackNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.DISCORD && (
          <EditDiscordNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}
        {notifier?.notifierType === NotifierType.TEAMS && (
          <EditTeamsNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.MATTERMOST && (
          <EditMattermostNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.SES && (
          <EditSESNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.SNS && (
          <EditSNSNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}
      </div>

      <div className="mt-3 flex">
        {isUnsaved && !isTestNotificationSuccess ? (
          <Button
            className="mr-1"
            disabled={isSendingTestNotification || !isAllDataFilled()}
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
            disabled={isSaving || !isAllDataFilled()}
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
