import { App, Button } from 'antd';
import { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { settingsApi } from '../../../entity/users';
import { type ApiComponents } from '../../../shared/api';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../../shared/i18n';

interface Props {
  isEmailConfigured: boolean;
}

export function MailServerComponent({ isEmailConfigured }: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();
  const [isSendingTestEmail, setIsSendingTestEmail] = useState(false);
  const [testEmailError, setTestEmailError] = useState<unknown>(undefined);
  const [undeliveredTestEmail, setUndeliveredTestEmail] = useState<
    ApiComponents['schemas']['TestEmail'] | undefined
  >(undefined);

  const sendTestEmail = async () => {
    setIsSendingTestEmail(true);
    setTestEmailError(undefined);
    setUndeliveredTestEmail(undefined);

    try {
      const testEmail = await settingsApi.sendTestEmail();

      if (testEmail.isDelivered) {
        message.success(
          t('settings.mailServer.testEmailSent', { email: testEmail.recipientEmail }),
        );
      } else {
        setUndeliveredTestEmail(testEmail);
      }
    } catch (error: unknown) {
      setTestEmailError(error);
    } finally {
      setIsSendingTestEmail(false);
    }
  };

  return (
    <section className="my-8 max-w-2xl text-sm">
      <h2 className="mb-3 text-xl font-bold dark:text-white">{t('settings.mailServer.title')}</h2>

      <div className="text-gray-900 dark:text-white">
        {isEmailConfigured ? (
          <Trans
            i18nKey="settings.mailServer.configured"
            components={{
              status: <span className="font-semibold text-green-600 dark:text-green-400" />,
            }}
          />
        ) : (
          <Trans
            i18nKey="settings.mailServer.notConfigured"
            components={{
              status: <span className="font-semibold text-amber-500 dark:text-amber-400" />,
            }}
          />
        )}
      </div>

      <div className="mt-1 text-gray-500 dark:text-gray-400">
        {t('settings.mailServer.description')}
      </div>

      <div className="mt-1 text-gray-500 dark:text-gray-400">
        <Trans
          i18nKey="settings.mailServer.docs"
          components={{
            docsLink: (
              <a
                href={getWebsitePageUrl('advancedConfigEmailSmtp', locale)}
                target="_blank"
                rel="noreferrer"
                className="!text-blue-600"
              />
            ),
          }}
        />
      </div>

      <Button
        className="mt-3"
        type="primary"
        ghost
        size="small"
        onClick={sendTestEmail}
        loading={isSendingTestEmail}
        disabled={!isEmailConfigured || isSendingTestEmail}
      >
        {t('settings.mailServer.sendTestEmail')}
      </Button>

      {testEmailError !== undefined && (
        <div className="mt-2 break-words text-red-600">{translateApiError(testEmailError, t)}</div>
      )}

      {undeliveredTestEmail && (
        <div className="mt-2 break-words text-red-600">
          <Trans
            i18nKey="settings.mailServer.testEmailNotDelivered"
            components={{
              email: <span>{undeliveredTestEmail.recipientEmail}</span>,
              failureDetail: <span>{undeliveredTestEmail.failureDetail}</span>,
            }}
          />
        </div>
      )}
    </section>
  );
}
