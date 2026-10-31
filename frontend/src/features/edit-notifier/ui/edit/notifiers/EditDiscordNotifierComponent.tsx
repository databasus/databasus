import { Input } from 'antd';
import { useTranslation } from 'react-i18next';

import type { DiscordNotifierSettings } from '../../../../../entities/notifiers';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: DiscordNotifierSettings;
  isSaved: boolean;
  onChange: (settings: DiscordNotifierSettings) => void;
}

export function EditDiscordNotifierComponent({ settings, isSaved, onChange }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('notifiers.discord.channelWebhookUrl')}
        </div>
        <Input
          aria-label={t('notifiers.discord.channelWebhookUrl')}
          value={settings.channelWebhookUrl ?? ''}
          onChange={(e) =>
            onChange({
              ...settings,
              channelWebhookUrl: toOptionalValue(e.target.value.trim()),
            })
          }
          size="small"
          className="w-full max-w-[250px]"
          placeholder={
            isSaved
              ? t('notifiers.secretKeptPlaceholder')
              : // eslint-disable-next-line i18next/no-literal-string -- example URL, not copy
                'https://discord.com/api/webhooks/...'
          }
        />
      </div>

      <div className="max-w-[250px] sm:ml-[150px]">
        <div className="mt-1 text-xs text-muted">
          <strong>{t('notifiers.discord.howTo.title')}</strong>
          <br />
          <br />
          {t('notifiers.discord.howTo.createChannel')}
          <br />
          {t('notifiers.discord.howTo.openChannelSettings')}
          <br />
          {t('notifiers.discord.howTo.openIntegrations')}
          <br />
          {t('notifiers.discord.howTo.createWebhook')}
          <br />
          {t('notifiers.discord.howTo.copyWebhookUrl')}
          <br />
          <br />
          <em>{t('notifiers.discord.howTo.privacyNote')}</em>
        </div>
      </div>
    </>
  );
}
