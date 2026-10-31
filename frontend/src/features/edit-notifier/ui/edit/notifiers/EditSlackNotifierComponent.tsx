import { Input } from 'antd';
import { useTranslation } from 'react-i18next';

import type { SlackNotifierSettings } from '../../../../../entities/notifiers';
import { getWebsitePageUrl, useLocale } from '../../../../../shared/i18n';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: SlackNotifierSettings;
  isSaved: boolean;
  onChange: (settings: SlackNotifierSettings) => void;
}

export function EditSlackNotifierComponent({ settings, isSaved, onChange }: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();

  return (
    <>
      <div className="mb-1 max-w-[250px] sm:ml-[150px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-accent"
          href={getWebsitePageUrl('notifiersSlack', locale)}
          target="_blank"
          rel="noreferrer"
        >
          {t('notifiers.slack.howToConnect')}
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.fields.botToken')}</div>
        <Input
          aria-label={t('notifiers.fields.botToken')}
          value={settings.botToken ?? ''}
          onChange={(e) =>
            onChange({
              ...settings,
              botToken: toOptionalValue(e.target.value.trim()),
            })
          }
          size="small"
          className="w-full max-w-[250px]"
          placeholder={
            isSaved
              ? t('notifiers.secretKeptPlaceholder')
              : // eslint-disable-next-line i18next/no-literal-string -- example token prefix, not copy
                'xoxb-...'
          }
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('notifiers.fields.targetChatId')}
        </div>
        <Input
          aria-label={t('notifiers.fields.targetChatId')}
          value={settings.targetChatId}
          onChange={(e) => onChange({ ...settings, targetChatId: e.target.value.trim() })}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example channel ID
          placeholder="C1234567890"
        />
      </div>
    </>
  );
}
