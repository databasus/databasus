import { useTranslation } from 'react-i18next';

import type { WebhookNotifierSettings } from '../../../../../entities/notifiers';
import { NOTIFICATION_CATEGORY_LABEL_KEYS, WebhookMethod } from '../../../../../entities/notifiers';

interface Props {
  settings: WebhookNotifierSettings;
}

export function ShowWebhookNotifierComponent({ settings }: Props) {
  const { t } = useTranslation();
  const headers = settings.headers.filter((header) => header.name);

  return (
    <>
      <div className="flex items-center">
        <div className="min-w-27.5 pr-2">{t('notifiers.fields.webhookUrl')}</div>
        <div className="max-w-87.5 truncate">{settings.webhookUrl}</div>
      </div>

      <div className="mt-1 mb-1 flex items-center">
        <div className="min-w-27.5 pr-2">{t('notifiers.webhook.method')}</div>
        <div>{settings.webhookMethod}</div>
      </div>

      <div className="mt-1 mb-1 flex items-center">
        <div className="min-w-27.5 pr-2">{t('notifiers.webhook.sendOn')}</div>
        <div>
          {settings.acceptedCategories
            .map((category) => t(NOTIFICATION_CATEGORY_LABEL_KEYS[category]))
            .join(t('common.listSeparator')) || '-'}
        </div>
      </div>

      {headers.length > 0 && (
        <div className="mt-1 mb-1 flex items-start">
          <div className="min-w-27.5 pr-2">{t('notifiers.webhook.headers')}</div>
          <div className="flex flex-col text-sm">
            {headers.map((header, i) => (
              <div key={i} className="text-muted">
                <span className="font-medium">{header.name}:</span>{' '}
                {t('notifiers.webhook.hiddenHeaderValue')}
              </div>
            ))}
          </div>
        </div>
      )}

      {settings.webhookMethod === WebhookMethod.POST && settings.bodyTemplate && (
        <div className="mt-1 mb-1 flex items-start">
          <div className="min-w-27.5 pr-2">{t('notifiers.webhook.bodyTemplate.title')}</div>
          <div className="max-w-87.5 rounded bg-surface p-2 font-mono text-xs whitespace-pre-wrap">
            {settings.bodyTemplate}
          </div>
        </div>
      )}
    </>
  );
}
