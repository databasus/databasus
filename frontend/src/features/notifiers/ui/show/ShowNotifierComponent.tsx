import { useTranslation } from 'react-i18next';

import {
  NOTIFIER_TYPE_LABEL_KEYS,
  type Notifier,
  NotifierType,
  getNotifierLogoFromType,
} from '../../../../entities/notifiers';
import { ShowDiscordNotifierComponent } from './notifier/ShowDiscordNotifierComponent';
import { ShowEmailNotifierComponent } from './notifier/ShowEmailNotifierComponent';
import { ShowMattermostNotifierComponent } from './notifier/ShowMattermostNotifierComponent';
import { ShowSlackNotifierComponent } from './notifier/ShowSlackNotifierComponent';
import { ShowTeamsNotifierComponent } from './notifier/ShowTeamsNotifierComponent';
import { ShowTelegramNotifierComponent } from './notifier/ShowTelegramNotifierComponent';
import { ShowWebhookNotifierComponent } from './notifier/ShowWebhookNotifierComponent';

interface Props {
  notifier: Notifier;
}

export function ShowNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation();
  const { settings } = notifier;

  return (
    <div>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('common.fields.type')}</div>

        {t(NOTIFIER_TYPE_LABEL_KEYS[settings.type])}
        <img src={getNotifierLogoFromType(settings.type)} className="ml-1 h-4 w-4" />
      </div>

      <div>
        {settings.type === NotifierType.TELEGRAM && (
          <ShowTelegramNotifierComponent settings={settings} />
        )}

        {settings.type === NotifierType.EMAIL && <ShowEmailNotifierComponent settings={settings} />}

        {settings.type === NotifierType.WEBHOOK && (
          <ShowWebhookNotifierComponent settings={settings} />
        )}

        {settings.type === NotifierType.SLACK && <ShowSlackNotifierComponent settings={settings} />}

        {settings.type === NotifierType.DISCORD && <ShowDiscordNotifierComponent />}

        {settings.type === NotifierType.TEAMS && <ShowTeamsNotifierComponent />}

        {settings.type === NotifierType.MATTERMOST && (
          <ShowMattermostNotifierComponent settings={settings} />
        )}
      </div>
    </div>
  );
}
