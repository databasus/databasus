import { useTranslation } from 'react-i18next';

import type { SlackNotifierSettings } from '../../../../../entities/notifiers';

interface Props {
  settings: SlackNotifierSettings;
}

export function ShowSlackNotifierComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="flex items-center">
        <div className="min-w-27.5 pr-2">{t('notifiers.fields.botToken')}</div>

        <div>*********</div>
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('notifiers.fields.targetChatId')}</div>
        {settings.targetChatId}
      </div>
    </>
  );
}
