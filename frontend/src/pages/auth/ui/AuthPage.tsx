import { Alert, Loader } from '@mantine/core';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { isRestoredSignInExpected } from '@/entities/instance-backups';
import { type PendingSignIn, type SignInAuthenticatorSetup, userApi } from '@/entities/users';
import { RequestResetPasswordComponent } from '@/pages/auth/ui/RequestResetPasswordComponent';
import { ResetPasswordComponent } from '@/pages/auth/ui/ResetPasswordComponent';
import { SignInAuthenticatorCodeComponent } from '@/pages/auth/ui/SignInAuthenticatorCodeComponent';
import { SignInAuthenticatorSetupComponent } from '@/pages/auth/ui/SignInAuthenticatorSetupComponent';
import { SignInCodeComponent } from '@/pages/auth/ui/SignInCodeComponent';
import { SignInComponent } from '@/pages/auth/ui/SignInComponent';
import { SignUpComponent } from '@/pages/auth/ui/SignUpComponent';
import { useSessionQuery } from '@/shared/api';
import { beginAuthIntent, abandonAuthIntent } from '@/shared/auth';
import { useScreenHeight } from '@/shared/hooks';
import { BrandNavbar, LoadFailure } from '@/shared/ui';

export function AuthPage() {
  useEffect(() => {
    const intent = beginAuthIntent();
    return () => abandonAuthIntent(intent);
  }, []);
  const query = useSessionQuery({
    queryKey: ['registration-options'],
    queryFn: (signal) => userApi.isAnyUserExists({ signal }),
  });
  const hasAnyUser = query.data ?? true;
  const [authMode, setAuthMode] = useState<
    | 'signIn'
    | 'signUp'
    | 'requestReset'
    | 'resetPassword'
    | 'signInCode'
    | 'signInAuthenticatorCode'
    | 'signInAuthenticatorSetup'
  >('signIn');
  const [resetEmail, setResetEmail] = useState('');
  const [pendingSignIn, setPendingSignIn] = useState<PendingSignIn | undefined>(undefined);
  const [authenticatorPendingSignInId, setAuthenticatorPendingSignInId] = useState<string>();
  const [authenticatorSetup, setAuthenticatorSetup] = useState<SignInAuthenticatorSetup>();
  const isLoading = query.isPending;
  const failure = query.error ?? undefined;

  const returnToPasswordStep = () => {
    setPendingSignIn(undefined);
    setAuthenticatorPendingSignInId(undefined);
    setAuthenticatorSetup(undefined);
    setAuthMode('signIn');
  };

  const loadOptions = () => query.refetch();
  const screenHeight = useScreenHeight();
  const { t } = useTranslation();
  const [isRestoredSignIn] = useState(isRestoredSignInExpected);
  useEffect(() => {
    if (query.data !== undefined) setAuthMode(query.data ? 'signIn' : 'signUp');
  }, [query.data]);

  return (
    <div
      className="flex min-h-0 min-w-0 flex-col overflow-y-auto bg-page text-content"
      style={{ height: screenHeight }}
    >
      {isLoading ? (
        <div className="flex h-screen w-screen items-center justify-center">
          <Loader size="md" role="status" />
        </div>
      ) : failure !== undefined ? (
        <LoadFailure className="m-4" error={failure} onRetry={() => void loadOptions()} />
      ) : (
        <div>
          <div>
            <BrandNavbar />

            {isRestoredSignIn && (
              <div className="mx-auto mt-6 max-w-md px-4">
                <Alert color="accent" title={t('instanceRecovery.restoredSignIn')} />
              </div>
            )}

            <div className="mt-8 flex justify-center px-4 pb-8 sm:mt-[6vh]">
              {authMode === 'signUp' ? (
                <SignUpComponent
                  onSwitchToSignIn={hasAnyUser ? () => setAuthMode('signIn') : undefined}
                  isClaimingInstance={!hasAnyUser}
                />
              ) : authMode === 'signIn' ? (
                <SignInComponent
                  onSwitchToSignUp={() => setAuthMode('signUp')}
                  onSwitchToResetPassword={() => setAuthMode('requestReset')}
                  onCodeRequired={(newPendingSignIn) => {
                    setPendingSignIn(newPendingSignIn);
                    setAuthMode('signInCode');
                  }}
                  onAuthenticatorCodeRequired={(pendingSignInId) => {
                    setAuthenticatorPendingSignInId(pendingSignInId);
                    setAuthMode('signInAuthenticatorCode');
                  }}
                  onAuthenticatorSetupRequired={(setup) => {
                    setAuthenticatorSetup(setup);
                    setAuthMode('signInAuthenticatorSetup');
                  }}
                />
              ) : authMode === 'signInCode' && pendingSignIn ? (
                <SignInCodeComponent
                  pendingSignIn={pendingSignIn}
                  onPendingSignInReplaced={setPendingSignIn}
                  onPendingSignInLost={returnToPasswordStep}
                  onSwitchToSignIn={returnToPasswordStep}
                />
              ) : authMode === 'signInAuthenticatorCode' && authenticatorPendingSignInId ? (
                <SignInAuthenticatorCodeComponent
                  pendingSignInId={authenticatorPendingSignInId}
                  onPendingSignInLost={returnToPasswordStep}
                  onSwitchToSignIn={returnToPasswordStep}
                />
              ) : authMode === 'signInAuthenticatorSetup' && authenticatorSetup ? (
                <SignInAuthenticatorSetupComponent
                  setup={authenticatorSetup}
                  onPendingSignInLost={returnToPasswordStep}
                  onSwitchToSignIn={returnToPasswordStep}
                />
              ) : authMode === 'requestReset' ? (
                <RequestResetPasswordComponent
                  onSwitchToSignIn={() => setAuthMode('signIn')}
                  onSwitchToResetPassword={(email) => {
                    setResetEmail(email);
                    setAuthMode('resetPassword');
                  }}
                />
              ) : (
                <ResetPasswordComponent
                  onSwitchToSignIn={() => setAuthMode('signIn')}
                  onSwitchToRequestCode={() => setAuthMode('requestReset')}
                  initialEmail={resetEmail}
                />
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
