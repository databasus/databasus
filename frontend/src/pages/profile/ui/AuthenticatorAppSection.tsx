import { Button } from '@mantine/core';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type User, userApi } from '@/entities/users';
import type { AuthenticatorAppAction } from '@/pages/profile/model/AuthenticatorAppAction';
import { AuthenticatorAppDialog } from '@/pages/profile/ui/AuthenticatorAppDialog';
import { useSessionGuard } from '@/shared/auth';
import { useLocale } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';

interface Props {
  user: User;
  isAuthenticatorAppRequired: boolean;
  isSetupRequested?: boolean;
  onUserChanged: (user: User) => void;
  onSetupOpened?: () => void;
}

export function AuthenticatorAppSection({
  user,
  isAuthenticatorAppRequired,
  isSetupRequested = false,
  onUserChanged,
  onSetupOpened,
}: Props) {
  const { t, i18n } = useTranslation();
  const { formatDateTimeWithRelativeTime } = useLocale();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const isConnected = user.authenticatorAppConnectedAt !== null;
  const [openAction, setOpenAction] = useState<AuthenticatorAppAction | undefined>(() =>
    isSetupRequested && !isConnected ? 'connect' : undefined,
  );

  useEffect(() => {
    if (isSetupRequested) onSetupOpened?.();
  }, [isSetupRequested, onSetupOpened]);

  // The answers that change the app carry only the recovery codes, so the connection date and the
  // remaining count come from the account itself.
  const reloadUser = async () => {
    try {
      const reloadedUser = await userApi.getCurrentUser();
      if (isCurrent()) onUserChanged(reloadedUser);
    } catch {
      // The section keeps the previous state; the next profile load shows the new one.
    }
  };

  return (
    <div className="mb-10 max-w-md">
      <h3 className="mb-2 text-lg font-semibold">{t('users.profile.authenticatorApp.title')}</h3>
      <p className="text-sm text-muted">{t('users.profile.authenticatorApp.description')}</p>

      <div className="mt-3 text-sm">
        {user.authenticatorAppConnectedAt !== null ? (
          <>
            <p>
              {t('users.profile.authenticatorApp.connectedAt', {
                time: formatDateTimeWithRelativeTime(user.authenticatorAppConnectedAt),
              })}
            </p>
            <p>
              {t('users.profile.authenticatorApp.recoveryCodesLeft', {
                count: user.unusedRecoveryCodeCount ?? 0,
              })}
            </p>
          </>
        ) : (
          <p>{t('users.profile.authenticatorApp.notConnected')}</p>
        )}
      </div>

      <div className="mt-3 flex flex-wrap gap-2">
        {isConnected ? (
          <>
            <Button size="xs" variant="default" onClick={() => setOpenAction('replace')}>
              {t('users.profile.authenticatorApp.replace')}
            </Button>
            <Button size="xs" variant="default" onClick={() => setOpenAction('regenerate')}>
              {t('users.profile.authenticatorApp.regenerate')}
            </Button>
            {!isAuthenticatorAppRequired && (
              <Button
                size="xs"
                variant="outline"
                color="danger"
                onClick={() => setOpenAction('disconnect')}
              >
                {t('users.profile.authenticatorApp.disconnect')}
              </Button>
            )}
          </>
        ) : (
          <Button size="xs" variant="filled" onClick={() => setOpenAction('connect')}>
            {t('users.profile.authenticatorApp.connect')}
          </Button>
        )}
      </div>

      {isConnected && isAuthenticatorAppRequired && (
        <p className="mt-2 text-sm text-muted">
          {t('users.profile.authenticatorApp.requiredByInstance')}
        </p>
      )}

      {openAction && (
        <AuthenticatorAppDialog
          action={openAction}
          onAppChanged={() => {
            if (openAction === 'connect' || openAction === 'replace') {
              notify({
                kind: 'success',
                title: i18n.t('users.profile.authenticatorApp.connected'),
              });
            }
            void reloadUser();
          }}
          onDisconnected={(disconnectedUser) => {
            setOpenAction(undefined);
            onUserChanged(disconnectedUser);
            notify({
              kind: 'success',
              title: i18n.t('users.profile.authenticatorApp.disconnected'),
            });
          }}
          onClose={() => setOpenAction(undefined)}
        />
      )}
    </div>
  );
}
