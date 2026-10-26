import { useTranslation } from 'react-i18next';

import type { EmailNotifierSettings } from '../../../../../entity/notifiers';

interface Props {
  settings: EmailNotifierSettings;
}

export function ShowEmailNotifierComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.email.targetEmail')}</div>
        {settings.targetEmail}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.email.smtpHost')}</div>
        {settings.smtpHost}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.email.smtpPort')}</div>
        {settings.smtpPort}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.email.smtpUser')}</div>
        {settings.smtpUser}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.email.smtpPassword')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.email.from')}</div>
        {settings.from || t('notifiers.email.fromAuto')}
      </div>

      {settings.isInsecureSkipVerify && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('notifiers.fields.skipTls')}</div>
          {t('common.states.enabled')}
        </div>
      )}
    </>
  );
}
