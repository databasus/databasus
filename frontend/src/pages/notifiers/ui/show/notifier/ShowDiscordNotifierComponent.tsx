import { useTranslation } from 'react-i18next';

export function ShowDiscordNotifierComponent() {
  const { t } = useTranslation();

  return (
    <div className="flex">
      <div className="max-w-27.5 min-w-27.5 pr-3">{t('notifiers.discord.channelWebhookUrl')}</div>

      <div>*********</div>
    </div>
  );
}
