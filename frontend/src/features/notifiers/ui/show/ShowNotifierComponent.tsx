import { useTranslation } from 'react-i18next';

import {
  NOTIFIER_TYPE_LABEL_KEYS,
  type Notifier,
  NotifierType,
} from '../../../../entity/notifiers';
import { getNotifierLogoFromType } from '../../../../entity/notifiers/models/getNotifierLogoFromType';
import { ShowDiscordNotifierComponent } from './notifier/ShowDiscordNotifierComponent';
import { ShowEmailNotifierComponent } from './notifier/ShowEmailNotifierComponent';
import { ShowMattermostNotifierComponent } from './notifier/ShowMattermostNotifierComponent';
import { ShowSESNotifierComponent } from './notifier/ShowSESNotifierComponent';
import { ShowSlackNotifierComponent } from './notifier/ShowSlackNotifierComponent';
import { ShowSNSNotifierComponent } from './notifier/ShowSNSNotifierComponent';
import { ShowTeamsNotifierComponent } from './notifier/ShowTeamsNotifierComponent';
import { ShowTelegramNotifierComponent } from './notifier/ShowTelegramNotifierComponent';
import { ShowWebhookNotifierComponent } from './notifier/ShowWebhookNotifierComponent';

interface Props {
  notifier: Notifier;
}

export function ShowNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation();

  return (
    <div>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('common.fields.type')}</div>

        {t(NOTIFIER_TYPE_LABEL_KEYS[notifier.notifierType])}
        <img src={getNotifierLogoFromType(notifier?.notifierType)} className="ml-1 h-4 w-4" />
      </div>

      <div>
        {notifier?.notifierType === NotifierType.TELEGRAM && (
          <ShowTelegramNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.EMAIL && (
          <ShowEmailNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.WEBHOOK && (
          <ShowWebhookNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.SLACK && (
          <ShowSlackNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.DISCORD && (
          <ShowDiscordNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.TEAMS && (
          <ShowTeamsNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.MATTERMOST && (
          <ShowMattermostNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.SES && (
          <ShowSESNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.SNS && (
          <ShowSNSNotifierComponent notifier={notifier} />
        )}
      </div>
    </div>
  );
}
