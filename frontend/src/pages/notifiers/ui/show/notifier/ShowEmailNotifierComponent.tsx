import { useTranslation } from 'react-i18next';

import type { EmailNotifierSettings } from '@/entities/notifiers';

interface Props {
  settings: EmailNotifierSettings;
}

export function ShowEmailNotifierComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('notifiers.email.targetEmail')}</div>
        {settings.targetEmail}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('notifiers.email.smtpHost')}</div>
        {settings.smtpHost}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('notifiers.email.smtpPort')}</div>
        {settings.smtpPort}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('notifiers.email.smtpUser')}</div>
        {settings.smtpUser}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('notifiers.email.smtpPassword')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('notifiers.email.from')}</div>
        {settings.from || t('notifiers.email.fromAuto')}
      </div>

      {settings.isInsecureSkipVerify && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('notifiers.fields.skipTls')}</div>
          {t('common.states.enabled')}
        </div>
      )}
    </>
  );
}
