import { Button } from '@mantine/core';
import { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { settingsApi } from '@/entities/users';
import { useSessionMutation } from '@/shared/api';
import { type ApiComponents } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
interface Props {
  isEmailConfigured: boolean;
}

export function MailServer({ isEmailConfigured }: Props) {
  const sendTestEmailMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof settingsApi.sendTestEmail>) =>
      settingsApi.sendTestEmail(...parameters),
  });

  const isSendingTestEmail = sendTestEmailMutation.isPending;
  const [testEmailError, setTestEmailError] = useState<unknown>(undefined);
  const [undeliveredTestEmail, setUndeliveredTestEmail] = useState<
    ApiComponents['schemas']['TestEmail'] | undefined
  >(undefined);

  const sendTestEmail = async () => {
    if (isSendingTestEmail) return;

    setTestEmailError(undefined);
    setUndeliveredTestEmail(undefined);

    try {
      const testEmail = await sendTestEmailMutation.mutateAsync([]);

      if (!isCurrent()) return;
      if (testEmail.isDelivered) {
        notify({
          kind: 'success',
          title: i18n.t('settings.mailServer.testEmailSent', { email: testEmail.recipientEmail }),
        });
      } else {
        setUndeliveredTestEmail(testEmail);
      }
    } catch (error: unknown) {
      if (isCurrent()) setTestEmailError(error);
    } finally {
      if (isCurrent()) sendTestEmailMutation.reset();
    }
  };
  const isCurrent = useSessionGuard();
  const { t, i18n } = useTranslation();
  const { locale } = useLocale();
  const notify = useFeedback();
  return (
    <section className="my-8 max-w-2xl text-sm">
      <h2 className="mb-3 text-xl font-bold">{t('settings.mailServer.title')}</h2>

      <div className="text-content">
        {isEmailConfigured ? (
          <Trans
            i18nKey="settings.mailServer.configured"
            components={{
              status: <span className="font-semibold text-success" />,
            }}
          />
        ) : (
          <Trans
            i18nKey="settings.mailServer.notConfigured"
            components={{
              status: <span className="font-semibold text-muted" />,
            }}
          />
        )}
      </div>

      <div className="mt-1 text-muted">{t('settings.mailServer.description')}</div>

      <div className="mt-1 text-muted">
        <Trans
          i18nKey="settings.mailServer.docs"
          components={{
            docsLink: (
              <a
                href={getWebsitePageUrl('advancedConfigEmailSmtp', locale)}
                target="_blank"
                rel="noreferrer"
                className="!text-accent"
              />
            ),
          }}
        />
      </div>

      <Button
        className="mt-3 !h-auto min-h-8 max-w-full !whitespace-normal"
        variant="outline"
        size="xs"
        onClick={sendTestEmail}
        loading={isSendingTestEmail}
        disabled={!isEmailConfigured || isSendingTestEmail}
        aria-busy={isSendingTestEmail}
      >
        {t('settings.mailServer.sendTestEmail')}
      </Button>

      {testEmailError !== undefined && (
        <div className="mt-2 break-words text-danger">{translateApiError(testEmailError, t)}</div>
      )}

      {undeliveredTestEmail && (
        <div className="mt-2 break-words text-danger">
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
