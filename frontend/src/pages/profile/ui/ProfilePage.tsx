import { EyeInvisibleOutlined, EyeTwoTone, LoadingOutlined } from '@ant-design/icons';
import { Form, Alert, App, Button, Input, Spin } from 'antd';
import { useEffectEvent, useId } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { USER_ROLE_LABEL_KEYS, type User, userApi } from '@/entities/users';
import { invalidateSession, useSessionGuard } from '@/shared/auth';
import { useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { LoadFailure } from '@/shared/ui';

interface Props {
  onUserChanged?: (user: User) => void;
}

const MIN_PASSWORD_LENGTH = 8;

export function ProfilePage({ onUserChanged }: Props) {
  const [failure, setFailure] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [user, setUser] = useState<User | undefined>(undefined);
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  const [editName, setEditName] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const [isUpdatingProfile, setIsUpdatingProfile] = useState(false);
  const [isEditNameError, setIsEditNameError] = useState(false);
  const [isEditEmailError, setIsEditEmailError] = useState(false);

  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isNewPasswordVisible, setIsNewPasswordVisible] = useState(false);
  const [isConfirmPasswordVisible, setIsConfirmPasswordVisible] = useState(false);

  const [isNewPasswordError, setIsNewPasswordError] = useState(false);
  const [isConfirmPasswordError, setIsConfirmPasswordError] = useState(false);

  const loadUserProfile = async () => {
    const revision = loadRevision.Invalidate();
    setIsLoading(true);
    setFailure(undefined);
    try {
      const loadedUser = await userApi.getCurrentUser();
      if (!isCurrent() || !loadRevision.IsCurrent(revision)) return;
      setUser(loadedUser);
      setEditName(loadedUser.name);
      setEditEmail(loadedUser.email);
    } catch (error) {
      if (isCurrent() && loadRevision.IsCurrent(revision)) setFailure(error);
    } finally {
      if (isCurrent() && loadRevision.IsCurrent(revision)) setIsLoading(false);
    }
  };

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

    setIsChangingPassword(true);

    try {
      await userApi.changePassword({ newPassword });
      if (!isCurrent()) return;

      setNewPassword('');
      setConfirmPassword('');
      invalidateSession();
      message.success(i18n.t('users.profile.changePassword.signInRequired'));
    } catch (error: unknown) {
      if (isCurrent()) setFailure(error);
    } finally {
      if (isCurrent()) setIsChangingPassword(false);
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

    setIsUpdatingProfile(true);

    try {
      if (editName === user?.name && editEmail === user?.email) {
        message.info(t('users.profile.noChanges'));
        if (isCurrent()) setIsUpdatingProfile(false);
        return;
      }

      const submitted = { name: editName, email: editEmail };
      const acknowledged = await userApi.updateUserInfo(submitted);
      if (!isCurrent()) return;

      setUser(acknowledged);
      setEditName((current) => (current === submitted.name ? acknowledged.name : current));
      setEditEmail((current) => (current === submitted.email ? acknowledged.email : current));
      onUserChanged?.(acknowledged);
      message.success(i18n.t('users.profile.updated'));
    } catch (error: unknown) {
      if (isCurrent()) setFailure(error);
    } finally {
      if (isCurrent()) setIsUpdatingProfile(false);
    }
  };

  const handleLogout = () => {
    invalidateSession();
  };
  const controlId = useId();
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const isCurrent = useSessionGuard();
  const loadRevision = useRequestRevision();

  const startProfileLoad = useEffectEvent(() => {
    void loadUserProfile();
  });

  useEffect(() => {
    startProfileLoad();
    return () => {
      loadRevision.Invalidate();
    };
  }, [loadRevision]);

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <div className="flex min-h-0 w-full min-w-0 flex-1 flex-col">
        <div className="min-h-0 min-w-0 flex-1 overflow-y-auto rounded bg-surface p-4 wrap-anywhere shadow sm:p-5">
          <h1 className="text-2xl font-bold">{t('users.profile.title')}</h1>

          {failure !== undefined &&
            (user ? (
              <Alert
                className="mt-3"
                type="error"
                showIcon
                message={translateApiError(failure, t)}
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

                    <Form.Item
                      htmlFor={controlId + 'field-1'}
                      label={t('users.fields.name')}
                      layout="vertical"
                      className="!mb-3"
                      validateStatus={isEditNameError ? 'error' : undefined}
                      help={
                        isEditNameError ? (
                          <span id={controlId + 'field-1' + '-error'}>
                            {t('users.validation.nameRequired')}
                          </span>
                        ) : undefined
                      }
                    >
                      <Input
                        size="small"
                        aria-invalid={isEditNameError}
                        aria-describedby={controlId + 'field-1' + '-error'}
                        id={controlId + 'field-1'}
                        aria-label={t('users.fields.name')}
                        value={editName}
                        onChange={(e) => {
                          setIsEditNameError(false);
                          setEditName(e.currentTarget.value);
                        }}
                        status={isEditNameError ? 'error' : undefined}
                        placeholder={t('users.profile.namePlaceholder')}
                      />
                    </Form.Item>

                    <Form.Item
                      htmlFor={controlId + 'field-2'}
                      label={t('users.fields.email')}
                      layout="vertical"
                      className="!mb-3"
                      validateStatus={isEditEmailError ? 'error' : undefined}
                      help={
                        isEditEmailError ? (
                          <span id={controlId + 'field-2' + '-error'}>
                            {t('users.validation.emailRequired')}
                          </span>
                        ) : undefined
                      }
                    >
                      <Input
                        size="small"
                        aria-invalid={isEditEmailError}
                        aria-describedby={controlId + 'field-2' + '-error'}
                        id={controlId + 'field-2'}
                        aria-label={t('users.fields.email')}
                        value={editEmail}
                        onChange={(e) => {
                          setIsEditEmailError(false);
                          setEditEmail(e.currentTarget.value.trim().toLowerCase());
                        }}
                        status={isEditEmailError ? 'error' : undefined}
                        placeholder={t('users.profile.emailPlaceholder')}
                        type="email"
                      />
                    </Form.Item>

                    <div className="mt-2 mb-1 text-xs font-semibold">{t('users.profile.role')}</div>
                    <div className="mb-4">
                      <span className="inline-flex items-center rounded-full bg-surface px-2.5 py-0.5 text-xs font-medium text-accent">
                        {t(USER_ROLE_LABEL_KEYS[user.role])}
                      </span>
                    </div>

                    {(editName !== user.name || editEmail !== user.email) && (
                      <Button
                        className="!h-auto min-h-8 max-w-full !whitespace-normal"
                        type="primary"
                        aria-label={t('users.profile.saveChanges')}
                        onClick={handleProfileUpdate}
                        loading={isUpdatingProfile}
                        disabled={isUpdatingProfile}
                      >
                        {t('users.profile.saveChanges')}
                      </Button>
                    )}
                  </div>
                </div>

                <div className="mb-6">
                  <Button
                    className="!h-auto min-h-8 max-w-full !whitespace-normal"
                    type="primary"
                    ghost
                    onClick={handleLogout}
                    danger
                  >
                    {t('users.profile.logout')}
                  </Button>
                </div>

                <div className="max-w-xs">
                  <h3 className="mb-4 text-lg font-semibold">
                    {t('users.profile.changePassword.title')}
                  </h3>

                  <div className="max-w-sm">
                    <Form.Item
                      htmlFor={controlId + 'field-3'}
                      label={t('users.fields.newPassword')}
                      layout="vertical"
                      className="!mb-3"
                      validateStatus={isNewPasswordError ? 'error' : undefined}
                      help={
                        isNewPasswordError ? (
                          <span id={controlId + 'field-3' + '-error'}>
                            {t('users.validation.passwordTooShort', {
                              minLength: MIN_PASSWORD_LENGTH,
                            })}
                          </span>
                        ) : undefined
                      }
                    >
                      <Input.Password
                        size="small"
                        aria-invalid={isNewPasswordError}
                        aria-describedby={controlId + 'field-3' + '-error'}
                        id={controlId + 'field-3'}
                        placeholder={t('users.profile.changePassword.newPasswordPlaceholder')}
                        aria-label={t('common.fields.password')}
                        value={newPassword}
                        onChange={(e) => {
                          setIsNewPasswordError(false);
                          setNewPassword(e.currentTarget.value);
                        }}
                        status={isNewPasswordError ? 'error' : undefined}
                        iconRender={(visible) =>
                          visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                        }
                        visibilityToggle={{
                          visible: isNewPasswordVisible,
                          onVisibleChange: setIsNewPasswordVisible,
                        }}
                        autoComplete="new-password"
                      />
                    </Form.Item>

                    <Form.Item
                      htmlFor={controlId + 'field-4'}
                      label={t('users.profile.changePassword.confirmPassword')}
                      layout="vertical"
                      className="!mb-3"
                      validateStatus={isConfirmPasswordError ? 'error' : undefined}
                      help={
                        isConfirmPasswordError ? (
                          <span id={controlId + 'field-4' + '-error'}>
                            {t('users.profile.changePassword.passwordsDoNotMatch')}
                          </span>
                        ) : undefined
                      }
                    >
                      <Input.Password
                        size="small"
                        aria-invalid={isConfirmPasswordError}
                        aria-describedby={controlId + 'field-4' + '-error'}
                        id={controlId + 'field-4'}
                        placeholder={t('users.profile.changePassword.confirmPasswordPlaceholder')}
                        aria-label={t('users.profile.changePassword.confirmPassword')}
                        value={confirmPassword}
                        onChange={(e) => {
                          setIsConfirmPasswordError(false);
                          setConfirmPassword(e.currentTarget.value);
                        }}
                        status={isConfirmPasswordError ? 'error' : undefined}
                        iconRender={(visible) =>
                          visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                        }
                        autoComplete="new-password"
                        visibilityToggle={{
                          visible: isConfirmPasswordVisible,
                          onVisibleChange: setIsConfirmPasswordVisible,
                        }}
                      />
                    </Form.Item>

                    {(newPassword || confirmPassword) && (
                      <Button
                        className="!h-auto min-h-8 max-w-full !whitespace-normal"
                        type="primary"
                        onClick={handlePasswordChange}
                        loading={isChangingPassword}
                        disabled={isChangingPassword}
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
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}
