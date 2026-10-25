import { EyeInvisibleOutlined, EyeTwoTone, LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { USER_ROLE_LABEL_KEYS, type User, userApi } from '../../../entity/users';
import { translateApiError } from '../../../shared/i18n';

interface Props {
  contentHeight: number;
}

const MIN_PASSWORD_LENGTH = 8;

export function ProfileComponent({ contentHeight }: Props) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [user, setUser] = useState<User | undefined>(undefined);
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  // Profile edit state
  const [editName, setEditName] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const [isUpdatingProfile, setIsUpdatingProfile] = useState(false);
  const [editNameError, setEditNameError] = useState(false);
  const [editEmailError, setEditEmailError] = useState(false);

  // Password change form state
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [newPasswordVisible, setNewPasswordVisible] = useState(false);
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  // Error states
  const [newPasswordError, setNewPasswordError] = useState(false);
  const [confirmPasswordError, setConfirmPasswordError] = useState(false);

  useEffect(() => {
    loadUserProfile();
  }, []);

  const loadUserProfile = () => {
    userApi
      .getCurrentUser()
      .then((user) => {
        setUser(user);
        setEditName(user.name);
        setEditEmail(user.email);
      })
      .catch((error) => {
        message.error(translateApiError(error, t));
      });
  };

  const validatePasswordFields = (): boolean => {
    let isValid = true;

    if (!newPassword) {
      setNewPasswordError(true);
      isValid = false;
    } else if (newPassword.length < MIN_PASSWORD_LENGTH) {
      setNewPasswordError(true);
      message.error(t('users.validation.passwordTooShort', { minLength: MIN_PASSWORD_LENGTH }));
      isValid = false;
    } else {
      setNewPasswordError(false);
    }

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      isValid = false;
    } else if (newPassword !== confirmPassword) {
      setConfirmPasswordError(true);
      message.error(t('users.profile.changePassword.passwordsDoNotMatch'));
      isValid = false;
    } else {
      setConfirmPasswordError(false);
    }

    return isValid;
  };

  const handlePasswordChange = async () => {
    if (!validatePasswordFields()) {
      return;
    }

    setIsChangingPassword(true);

    try {
      await userApi.changePassword({ newPassword });

      // Reset form fields
      setNewPassword('');
      setConfirmPassword('');

      // Sign in again with new password
      if (user?.email) {
        try {
          await userApi.signIn({ email: user.email, password: newPassword });
          message.success(t('users.profile.changePassword.signedInWithNewPassword'));
        } catch (signInError: unknown) {
          message.error(translateApiError(signInError, t));
          // If sign in fails, logout and redirect to login page
          userApi.logout();
          userApi.notifyAuthListeners();
          window.location.reload();
        }
      }
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsChangingPassword(false);
    }
  };

  const handleProfileUpdate = async () => {
    // Validate name
    if (!editName || editName.trim() === '') {
      setEditNameError(true);
      message.error(t('users.validation.nameRequired'));
      return;
    }
    setEditNameError(false);

    if (!editEmail || editEmail.trim() === '') {
      setEditEmailError(true);
      message.error(t('users.validation.emailRequired'));
      return;
    }
    setEditEmailError(false);

    setIsUpdatingProfile(true);

    try {
      if (editName === user?.name && editEmail === user?.email) {
        message.info(t('users.profile.noChanges'));
        setIsUpdatingProfile(false);
        return;
      }

      await userApi.updateUserInfo({ name: editName, email: editEmail });
      message.success(t('users.profile.updated'));

      // Reload user profile
      loadUserProfile();
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsUpdatingProfile(false);
    }
  };

  const handleLogout = () => {
    userApi.logout();
    window.location.reload();
  };

  return (
    <div className="flex grow">
      <div className="w-full">
        <div
          className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800"
          style={{ height: contentHeight }}
        >
          <h1 className="text-2xl font-bold dark:text-white">{t('users.profile.title')}</h1>

          <div className="mt-5">
            {user ? (
              <>
                <div className="mb-6">
                  <h3 className="mb-4 text-lg font-semibold dark:text-white">
                    {t('users.profile.information')}
                  </h3>
                  <div className="max-w-md">
                    <div className="text-xs font-semibold dark:text-gray-200">
                      {t('users.profile.userId')}
                    </div>
                    <div className="mb-4 text-sm text-gray-600 dark:text-gray-400">{user.id}</div>

                    <div className="mb-1 text-xs font-semibold dark:text-gray-200">
                      {t('users.fields.name')}
                    </div>
                    <Input
                      value={editName}
                      onChange={(e) => {
                        setEditNameError(false);
                        setEditName(e.currentTarget.value);
                      }}
                      status={editNameError ? 'error' : undefined}
                      placeholder={t('users.profile.namePlaceholder')}
                      className="mb-4"
                    />

                    <div className="mt-2 mb-1 text-xs font-semibold dark:text-gray-200">
                      {t('users.fields.email')}
                    </div>
                    <Input
                      value={editEmail}
                      onChange={(e) => {
                        setEditEmailError(false);
                        setEditEmail(e.currentTarget.value.trim().toLowerCase());
                      }}
                      status={editEmailError ? 'error' : undefined}
                      placeholder={t('users.profile.emailPlaceholder')}
                      type="email"
                      className="mb-4"
                    />

                    <div className="mt-2 mb-1 text-xs font-semibold dark:text-gray-200">
                      {t('users.profile.role')}
                    </div>
                    <div className="mb-4">
                      <span className="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                        {t(USER_ROLE_LABEL_KEYS[user.role])}
                      </span>
                    </div>

                    {(editName !== user.name || editEmail !== user.email) && (
                      <Button
                        type="primary"
                        onClick={handleProfileUpdate}
                        loading={isUpdatingProfile}
                        disabled={isUpdatingProfile}
                        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                      >
                        {t('users.profile.saveChanges')}
                      </Button>
                    )}
                  </div>
                </div>

                <div className="mb-8">
                  <Button type="primary" ghost onClick={handleLogout} danger>
                    {t('users.profile.logout')}
                  </Button>
                </div>

                <div className="max-w-xs">
                  <h3 className="mb-4 text-lg font-semibold dark:text-white">
                    {t('users.profile.changePassword.title')}
                  </h3>

                  <div className="max-w-sm">
                    <div className="my-1 text-xs font-semibold dark:text-gray-200">
                      {t('users.fields.newPassword')}
                    </div>
                    <Input.Password
                      placeholder={t('users.profile.changePassword.newPasswordPlaceholder')}
                      value={newPassword}
                      onChange={(e) => {
                        setNewPasswordError(false);
                        setNewPassword(e.currentTarget.value);
                      }}
                      status={newPasswordError ? 'error' : undefined}
                      iconRender={(visible) =>
                        visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                      }
                      visibilityToggle={{
                        visible: newPasswordVisible,
                        onVisibleChange: setNewPasswordVisible,
                      }}
                      autoComplete="new-password"
                    />

                    <div className="mt-2 mb-1 text-xs font-semibold dark:text-gray-200">
                      {t('users.profile.changePassword.confirmPassword')}
                    </div>
                    <Input.Password
                      placeholder={t('users.profile.changePassword.confirmPasswordPlaceholder')}
                      value={confirmPassword}
                      onChange={(e) => {
                        setConfirmPasswordError(false);
                        setConfirmPassword(e.currentTarget.value);
                      }}
                      status={confirmPasswordError ? 'error' : undefined}
                      iconRender={(visible) =>
                        visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                      }
                      autoComplete="new-password"
                      visibilityToggle={{
                        visible: confirmPasswordVisible,
                        onVisibleChange: setConfirmPasswordVisible,
                      }}
                    />

                    <div className="mt-3" />

                    {(newPassword || confirmPassword) && (
                      <Button
                        type="primary"
                        onClick={handlePasswordChange}
                        loading={isChangingPassword}
                        disabled={isChangingPassword}
                        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                      >
                        {isChangingPassword
                          ? t('users.profile.changePassword.changing')
                          : t('users.profile.changePassword.submit')}
                      </Button>
                    )}
                  </div>
                </div>
              </>
            ) : (
              <div>
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
