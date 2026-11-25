import { Alert, Button, Loader } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type AuthenticatorAppSetup,
  type User,
  authenticatorAppApi,
  AuthenticatorAppQrCode,
  RecoveryCodesList,
  SecondFactorCodeField,
  isAppCodeFormat,
} from '@/entities/users';
import type { AuthenticatorAppAction } from '@/pages/profile/model/AuthenticatorAppAction';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { type TranslationKey, translateApiError } from '@/shared/i18n';
import { Dialog } from '@/shared/ui';

interface Props {
  action: AuthenticatorAppAction;
  onAppChanged: () => void;
  onDisconnected: (user: User) => void;
  onClose: () => void;
}

type Step =
  | { kind: 'proof' }
  | { kind: 'starting' }
  | { kind: 'setup'; setup: AuthenticatorAppSetup }
  | { kind: 'recoveryCodes'; recoveryCodes: string[] };

const TITLE_KEYS: Record<AuthenticatorAppAction, TranslationKey> = {
  connect: 'users.profile.authenticatorApp.connectTitle',
  replace: 'users.profile.authenticatorApp.replaceTitle',
  regenerate: 'users.profile.authenticatorApp.regenerateTitle',
  disconnect: 'users.profile.authenticatorApp.disconnectTitle',
};

const PROOF_DESCRIPTION_KEYS: Record<Exclude<AuthenticatorAppAction, 'connect'>, TranslationKey> = {
  replace: 'users.profile.authenticatorApp.replaceDescription',
  regenerate: 'users.profile.authenticatorApp.regenerateDescription',
  disconnect: 'users.profile.authenticatorApp.disconnectDescription',
};

// Recovery codes live only in this dialog's state, so closing it discards them from the page.
export function AuthenticatorAppDialog({ action, onAppChanged, onDisconnected, onClose }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const isSetupRequestedRef = useRef(false);
  const [step, setStep] = useState<Step>(
    action === 'connect' ? { kind: 'starting' } : { kind: 'proof' },
  );
  const [code, setCode] = useState('');
  const [isRecoveryCode, setIsRecoveryCode] = useState(false);
  const [isCodeFormatError, setIsCodeFormatError] = useState(false);
  const [failure, setFailure] = useState<unknown>();

  const startSetupMutation = useSessionMutation({
    mutationFn: (currentCode?: string) => authenticatorAppApi.startSetup(currentCode),
  });
  const confirmSetupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof authenticatorAppApi.confirmSetup>) =>
      authenticatorAppApi.confirmSetup(...parameters),
  });
  const regenerateMutation = useSessionMutation({
    mutationFn: (currentCode: string) => authenticatorAppApi.regenerateRecoveryCodes(currentCode),
  });
  const disconnectMutation = useSessionMutation({
    mutationFn: (currentCode: string) => authenticatorAppApi.disconnect(currentCode),
  });
  const isPending =
    startSetupMutation.isPending ||
    confirmSetupMutation.isPending ||
    regenerateMutation.isPending ||
    disconnectMutation.isPending;

  const resetEntry = () => {
    setCode('');
    setIsCodeFormatError(false);
  };

  const refuse = (error: unknown) => {
    setFailure(error);
    resetEntry();
  };

  const startSetup = async (currentCode?: string) => {
    try {
      const setup = await startSetupMutation.mutateAsync(currentCode);
      if (!isCurrent()) return;
      resetEntry();
      setStep({ kind: 'setup', setup });
    } catch (error: unknown) {
      if (!isCurrent()) return;
      if (action === 'connect') setStep({ kind: 'proof' });
      refuse(error);
    }
  };

  const startSetupOnOpen = useEffectEvent(() => void startSetup());

  useEffect(() => {
    if (action !== 'connect' || isSetupRequestedRef.current) return;

    isSetupRequestedRef.current = true;
    startSetupOnOpen();
  }, [action]);

  const submitProof = async () => {
    if (isRecoveryCode ? !code : !isAppCodeFormat(code)) {
      setIsCodeFormatError(true);

      return;
    }

    if (action === 'connect' || action === 'replace') {
      await startSetup(action === 'replace' ? code : undefined);

      return;
    }

    try {
      if (action === 'regenerate') {
        const recoveryCodes = await regenerateMutation.mutateAsync(code);
        if (!isCurrent()) return;
        setStep({ kind: 'recoveryCodes', recoveryCodes });
        onAppChanged();

        return;
      }

      const disconnectedUser = await disconnectMutation.mutateAsync(code);
      if (isCurrent()) onDisconnected(disconnectedUser);
    } catch (error: unknown) {
      if (isCurrent()) refuse(error);
    }
  };

  const confirmSetup = async (setup: AuthenticatorAppSetup) => {
    if (!isAppCodeFormat(code)) {
      setIsCodeFormatError(true);

      return;
    }

    try {
      const recoveryCodes = await confirmSetupMutation.mutateAsync([
        setup.authenticatorAppSetupId,
        code,
      ]);
      if (!isCurrent()) return;
      setStep({ kind: 'recoveryCodes', recoveryCodes });
      onAppChanged();
    } catch (error: unknown) {
      if (isCurrent()) refuse(error);
    }
  };

  const submit = () => {
    if (isPending) return;
    setFailure(undefined);

    if (step.kind === 'proof') void submitProof();
    if (step.kind === 'setup') void confirmSetup(step.setup);
  };

  const codeField = (canUseRecoveryCode: boolean) => (
    <SecondFactorCodeField
      code={code}
      onCodeChange={(enteredCode) => {
        setIsCodeFormatError(false);
        setFailure(undefined);
        setCode(enteredCode);
      }}
      isRecoveryCode={canUseRecoveryCode && isRecoveryCode}
      onRecoveryCodeToggle={
        canUseRecoveryCode
          ? (isRecoveryCodeChosen) => {
              setIsRecoveryCode(isRecoveryCodeChosen);
              resetEntry();
              setFailure(undefined);
            }
          : undefined
      }
      error={isCodeFormatError ? t('authenticatorApp.code.invalidAppCode') : undefined}
      isDisabled={isPending}
    />
  );

  return (
    <Dialog opened title={t(TITLE_KEYS[action])} onClose={onClose}>
      {step.kind === 'starting' && (
        <div className="flex justify-center py-4">
          <Loader role="status" />
        </div>
      )}

      {step.kind === 'recoveryCodes' ? (
        <>
          <RecoveryCodesList recoveryCodes={step.recoveryCodes} />
          <div className="mt-5 flex justify-end">
            <Button variant="filled" onClick={onClose}>
              {t('users.profile.authenticatorApp.done')}
            </Button>
          </div>
        </>
      ) : (
        step.kind !== 'starting' && (
          <form
            className="text-sm"
            onSubmit={(event) => {
              event.preventDefault();
              submit();
            }}
          >
            {step.kind === 'proof' && action !== 'connect' && (
              <>
                <p>{t(PROOF_DESCRIPTION_KEYS[action])}</p>
                <p className="mt-2 mb-3">{t('users.profile.authenticatorApp.currentCode')}</p>
                {codeField(true)}
              </>
            )}

            {step.kind === 'setup' && (
              <>
                <AuthenticatorAppQrCode
                  otpauthUri={step.setup.otpauthUri}
                  secret={step.setup.secret}
                />
                <div className="mt-4">{codeField(false)}</div>
              </>
            )}

            {failure !== undefined && (
              <Alert
                className="mt-3"
                color="danger"
                icon={<CircleAlert aria-hidden size={16} />}
                title={translateApiError(failure, t)}
              />
            )}

            <div className="mt-5 flex flex-wrap justify-end gap-2">
              <Button disabled={isPending} onClick={onClose}>
                {t('common.actions.cancel')}
              </Button>
              {(step.kind === 'setup' || action !== 'connect') && (
                <Button
                  type="submit"
                  variant="filled"
                  color={action === 'disconnect' ? 'danger' : 'accent'}
                  loading={isPending}
                >
                  {step.kind === 'setup' || action !== 'disconnect'
                    ? t('users.profile.authenticatorApp.confirm')
                    : t('users.profile.authenticatorApp.disconnect')}
                </Button>
              )}
            </div>
          </form>
        )
      )}
    </Dialog>
  );
}
