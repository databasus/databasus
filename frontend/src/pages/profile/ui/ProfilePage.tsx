import { Alert, Button, Loader, PasswordInput, TextInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useId } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { USER_ROLE_LABEL_KEYS, type User, userApi } from '@/entities/users';
import { useSessionMutation } from '@/shared/api';
import { useSessionQuery } from '@/shared/api';
import { invalidateSession, useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { LoadFailure } from '@/shared/ui';

interface Props {
  onUserChanged?: (user: User) => void;
}

const MIN_PASSWORD_LENGTH = 8;

export function ProfilePage({ onUserChanged }: Props) {
  const changePasswordMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.changePassword>) =>
      userApi.changePassword(...parameters),
  });
  const updateUserInfoMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.updateUserInfo>) =>
      userApi.updateUserInfo(...parameters),
  });

  const query = useSessionQuery({
    queryKey: ['profile'],
    queryFn: (signal) => userApi.getCurrentUser({ signal }),
  });
  const [actionFailure, setFailure] = useState<unknown>();
  const failure = actionFailure ?? query.error ?? undefined;
  const isLoading = query.isPending;
  const [user, setUser] = useState<User | undefined>(undefined);
  const isChangingPassword = changePasswordMutation.isPending;

  const [editName, setEditName] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const isUpdatingProfile = updateUserInfoMutation.isPending;
  const [isEditNameError, setIsEditNameError] = useState(false);
  const [isEditEmailError, setIsEditEmailError] = useState(false);

  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isNewPasswordVisible, setIsNewPasswordVisible] = useState(false);
  const [isConfirmPasswordVisible, setIsConfirmPasswordVisible] = useState(false);

  const [isNewPasswordError, setIsNewPasswordError] = useState(false);
  const [isConfirmPasswordError, setIsConfirmPasswordError] = useState(false);

  const loadUserProfile = () => {
    setFailure(undefined);
    return query.refetch();
  };
  useEffect(() => {
    if (!query.data) return;

    setUser(query.data);
    setEditName(query.data.name);
    setEditEmail(query.data.email);
  }, [query.data]);

  const validatePasswordFields = (): boolean => {
    let isValid = true;

    if (!newPassword) {
      setIsNewPasswordError(true);
      isValid = false;
    } else if (newPassword.length < MIN_PASSWORD_LENGTH) {
      setIsNewPasswordError(true);

      isValid = false;
    } else {
      setIsNewPasswordError(false);
    }

    if (!confirmPassword) {
      setIsConfirmPasswordError(true);
      isValid = false;
    } else if (newPassword !== confirmPassword) {
      setIsConfirmPasswordError(true);

      isValid = false;
    } else {
      setIsConfirmPasswordError(false);
    }

    return isValid;
  };

  const handlePasswordChange = async () => {
    if (isChangingPassword || !validatePasswordFields()) {
      return;
    }

    try {
      await changePasswordMutation.mutateAsync([{ newPassword }]);
      if (!isCurrent()) return;

      setNewPassword('');
      setConfirmPassword('');
      invalidateSession();
      notify({
        kind: 'success',
        title: i18n.t('users.profile.changePassword.signInRequired'),
        isSessionTransition: true,
      });
    } catch (error: unknown) {
      if (isCurrent()) setFailure(error);
    } finally {
      if (isCurrent()) changePasswordMutation.reset();
    }
  };

  const handleProfileUpdate = async () => {
    if (isUpdatingProfile) return;
    setFailure(undefined);
    if (!editName || editName.trim() === '') {
      setIsEditNameError(true);

      return;
    }
    setIsEditNameError(false);

    if (!editEmail || editEmail.trim() === '') {
      setIsEditEmailError(true);

      return;
    }
    setIsEditEmailError(false);

    try {
      if (editName === user?.name && editEmail === user?.email) {
        notify({ kind: 'info', title: t('users.profile.noChanges') });
        if (isCurrent()) updateUserInfoMutation.reset();
        return;
      }

      const submitted = { name: editName, email: editEmail };
      const acknowledged = await updateUserInfoMutation.mutateAsync([submitted]);
      if (!isCurrent()) return;

      setUser(acknowledged);
      setEditName((current) => (current === submitted.name ? acknowledged.name : current));
      setEditEmail((current) => (current === submitted.email ? acknowledged.email : current));
      onUserChanged?.(acknowledged);
      notify({ kind: 'success', title: i18n.t('users.profile.updated') });
    } catch (error: unknown) {
      if (isCurrent()) setFailure(error);
    } finally {
      if (isCurrent()) updateUserInfoMutation.reset();
    }
  };

  const handleLogout = () => {
    invalidateSession();
  };
  const controlId = useId();
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <div className="flex min-h-0 w-full min-w-0 flex-1 flex-col">
        <div className="min-h-0 min-w-0 flex-1 overflow-y-auto rounded bg-surface p-4 wrap-anywhere shadow sm:p-5">
          <h1 className="text-2xl font-bold">{t('users.profile.title')}</h1>

          {failure !== undefined &&
            (user ? (
              <Alert
                className="mt-3"
                color="danger"
                icon={
                  <CircleAlert
                    aria-hidden
                    size={16}
                    className="inline-block shrink-0 align-middle"
                  />
                }
                title={translateApiError(failure, t)}
              />
            ) : (
              <LoadFailure
                className="mt-3"
                error={failure}
                onRetry={() => void loadUserProfile()}
              />
            ))}
          <div className="mt-5">
            {user ? (
              <>
                <div className="mb-4">
                  <h3 className="mb-4 text-lg font-semibold">{t('users.profile.information')}</h3>
                  <div className="max-w-xs">
                    <div className="text-xs font-semibold">{t('users.profile.userId')}</div>
                    <div className="mb-4 text-sm text-muted">{user.id}</div>

                    <div className="mb-3 min-w-0">
                      <label
                        htmlFor={controlId + 'field-1'}
                        className="mb-1 block text-sm text-content"
                      >
                        {t('users.fields.name')}
                      </label>
                      <TextInput
                        aria-invalid={isEditNameError}
                        id={controlId + 'field-1'}
                        aria-label={t('users.fields.name')}
                        value={editName}
                        onChange={(e) => {
                          setIsEditNameError(false);
                          setEditName(e.currentTarget.value);
                        }}
                        placeholder={t('users.profile.namePlaceholder')}
                        error={
                          isEditNameError ? (
                            <span>{t('users.validation.nameRequired')}</span>
                          ) : undefined
                        }
                      />
                    </div>

                    <div className="mb-3 min-w-0">
                      <label
                        htmlFor={controlId + 'field-2'}
                        className="mb-1 block text-sm text-content"
                      >
                        {t('users.fields.email')}
                      </label>
                      <TextInput
                        aria-invalid={isEditEmailError}
                        id={controlId + 'field-2'}
                        aria-label={t('users.fields.email')}
                        value={editEmail}
                        onChange={(e) => {
                          setIsEditEmailError(false);
                          setEditEmail(e.currentTarget.value.trim().toLowerCase());
                        }}
                        placeholder={t('users.profile.emailPlaceholder')}
                        type="email"
                        error={
                          isEditEmailError ? (
                            <span>{t('users.validation.emailRequired')}</span>
                          ) : undefined
                        }
                      />
                    </div>

                    <div className="mt-2 mb-1 text-xs font-semibold">{t('users.profile.role')}</div>
                    <div className="mb-4">
                      <span className="inline-flex items-center rounded-full bg-surface px-2.5 py-0.5 text-xs font-medium text-accent">
                        {t(USER_ROLE_LABEL_KEYS[user.role])}
                      </span>
                    </div>

                    {(editName !== user.name || editEmail !== user.email) && (
                      <Button
                        className="!h-auto min-h-8 max-w-full !whitespace-normal"
                        variant="filled"
                        aria-label={t('users.profile.saveChanges')}
                        onClick={handleProfileUpdate}
                        loading={isUpdatingProfile}
                        disabled={isUpdatingProfile}
                        aria-busy={isUpdatingProfile}
                      >
                        {t('users.profile.saveChanges')}
                      </Button>
                    )}
                  </div>
                </div>

                <div className="mb-6">
                  <Button
                    className="!h-auto min-h-8 max-w-full !whitespace-normal"
                    variant="outline"
                    onClick={handleLogout}
                    color="danger"
                  >
                    {t('users.profile.logout')}
                  </Button>
                </div>

                <div className="max-w-xs">
                  <h3 className="mb-4 text-lg font-semibold">
                    {t('users.profile.changePassword.title')}
                  </h3>

                  <div className="max-w-sm">
                    <div className="mb-3 min-w-0">
                      <label
                        htmlFor={controlId + 'field-3'}
                        className="mb-1 block text-sm text-content"
                      >
                        {t('users.fields.newPassword')}
                      </label>
                      <PasswordInput
                        aria-invalid={isNewPasswordError}
                        id={controlId + 'field-3'}
                        placeholder={t('users.profile.changePassword.newPasswordPlaceholder')}
                        aria-label={t('common.fields.password')}
                        value={newPassword}
                        onChange={(e) => {
                          setIsNewPasswordError(false);
                          setNewPassword(e.currentTarget.value);
                        }}
                        visible={isNewPasswordVisible}
                        onVisibilityChange={setIsNewPasswordVisible}
                        autoComplete="new-password"
                        visibilityToggleButtonProps={{
                          'aria-label': t('common.actions.togglePasswordVisibility'),
                        }}
                        error={
                          isNewPasswordError ? (
                            <span>
                              {t('users.validation.passwordTooShort', {
                                minLength: MIN_PASSWORD_LENGTH,
                              })}
                            </span>
                          ) : undefined
                        }
                      />
                    </div>

                    <div className="mb-3 min-w-0">
                      <label
                        htmlFor={controlId + 'field-4'}
                        className="mb-1 block text-sm text-content"
                      >
                        {t('users.profile.changePassword.confirmPassword')}
                      </label>
                      <PasswordInput
                        aria-invalid={isConfirmPasswordError}
                        id={controlId + 'field-4'}
                        placeholder={t('users.profile.changePassword.confirmPasswordPlaceholder')}
                        aria-label={t('users.profile.changePassword.confirmPassword')}
                        value={confirmPassword}
                        onChange={(e) => {
                          setIsConfirmPasswordError(false);
                          setConfirmPassword(e.currentTarget.value);
                        }}
                        autoComplete="new-password"
                        visible={isConfirmPasswordVisible}
                        onVisibilityChange={setIsConfirmPasswordVisible}
                        visibilityToggleButtonProps={{
                          'aria-label': t('common.actions.togglePasswordVisibility'),
                        }}
                        error={
                          isConfirmPasswordError ? (
                            <span>{t('users.profile.changePassword.passwordsDoNotMatch')}</span>
                          ) : undefined
                        }
                      />
                    </div>

                    {(newPassword || confirmPassword) && (
                      <Button
                        className="!h-auto min-h-8 max-w-full !whitespace-normal"
                        variant="filled"
                        onClick={handlePasswordChange}
                        loading={isChangingPassword}
                        disabled={isChangingPassword}
                        aria-busy={isChangingPassword}
                      >
                        {isChangingPassword
                          ? t('users.profile.changePassword.changing')
                          : t('users.profile.changePassword.submit')}
                      </Button>
                    )}
                  </div>
                </div>
              </>
            ) : isLoading ? (
              <div>
                <Loader role="status" />
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}
