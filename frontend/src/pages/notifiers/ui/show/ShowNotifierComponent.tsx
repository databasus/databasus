import { useTranslation } from 'react-i18next';

import {
  NOTIFIER_TYPE_LABEL_KEYS,
  type Notifier,
  NotifierType,
  getNotifierLogoFromType,
} from '@/entities/notifiers';
import { ShowDiscordNotifierComponent } from '@/pages/notifiers/ui/show/notifier/ShowDiscordNotifierComponent';
import { ShowEmailNotifierComponent } from '@/pages/notifiers/ui/show/notifier/ShowEmailNotifierComponent';
import { ShowMattermostNotifierComponent } from '@/pages/notifiers/ui/show/notifier/ShowMattermostNotifierComponent';
import { ShowSlackNotifierComponent } from '@/pages/notifiers/ui/show/notifier/ShowSlackNotifierComponent';
import { ShowTeamsNotifierComponent } from '@/pages/notifiers/ui/show/notifier/ShowTeamsNotifierComponent';
import { ShowTelegramNotifierComponent } from '@/pages/notifiers/ui/show/notifier/ShowTelegramNotifierComponent';
import { ShowWebhookNotifierComponent } from '@/pages/notifiers/ui/show/notifier/ShowWebhookNotifierComponent';

interface Props {
  notifier: Notifier;
}

export function ShowNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation();
  const { settings } = notifier;

  return (
    <div>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('common.fields.type')}</div>

        {t(NOTIFIER_TYPE_LABEL_KEYS[settings.type])}
        <img alt="" src={getNotifierLogoFromType(settings.type)} className="ml-1 h-4 w-4" />
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
