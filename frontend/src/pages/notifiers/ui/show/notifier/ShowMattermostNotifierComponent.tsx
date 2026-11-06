import { useTranslation } from 'react-i18next';

import {
  MATTERMOST_DELIVERY_MODE_LABEL_KEYS,
  MattermostDeliveryMode,
  type MattermostNotifierSettings,
} from '@/entities/notifiers';

interface Props {
  settings: MattermostNotifierSettings;
}

export function ShowMattermostNotifierComponent({ settings }: Props) {
  const { t } = useTranslation();
  const isWebhookMode = settings.deliveryMode === MattermostDeliveryMode.WEBHOOK;

  return (
    <>
      <div className="flex">
        <div className="max-w-27.5 min-w-27.5 pr-3">{t('notifiers.mattermost.connectVia')}</div>
        <div>{t(MATTERMOST_DELIVERY_MODE_LABEL_KEYS[settings.deliveryMode])}</div>
      </div>

      {isWebhookMode ? (
        <>
          <div className="flex">
            <div className="max-w-27.5 min-w-27.5 pr-3">{t('notifiers.fields.webhookUrl')}</div>
            <div className="break-all">*********</div>
          </div>

          {settings.targetChannelName && (
            <div className="flex">
              <div className="max-w-27.5 min-w-27.5 pr-3">{t('notifiers.mattermost.channel')}</div>
              <div className="break-all">{settings.targetChannelName}</div>
            </div>
          )}
        </>
      ) : (
        <>
          <div className="flex">
            <div className="max-w-27.5 min-w-27.5 pr-3">{t('notifiers.mattermost.serverUrl')}</div>
            <div className="break-all">{settings.serverUrl}</div>
          </div>

          <div className="flex">
            <div className="max-w-27.5 min-w-27.5 pr-3">{t('notifiers.fields.channelId')}</div>
            <div className="break-all">{settings.targetChannelId}</div>
          </div>
        </>
      )}
    </>
  );
}
