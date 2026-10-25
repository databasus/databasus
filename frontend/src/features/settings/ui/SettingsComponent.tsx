import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Spin, Switch } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { getApplicationServer } from '../../../constants';
import {
  type InstanceSettings,
  type InstanceSettingsUpdate,
  settingsApi,
} from '../../../entity/users';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../../shared/i18n';
import { ClipboardHelper } from '../../../shared/lib/ClipboardHelper';
import { VerificationAgentsComponent } from '../../verification/agents';
import { AuditLogsComponent } from './AuditLogsComponent';
import { MailServerComponent } from './MailServerComponent';

interface Props {
  contentHeight: number;
}

// The response carries the instance's answer about its mail server, which the
// form neither edits nor sends back.
const toFormSettings = (settings: InstanceSettings): InstanceSettingsUpdate => ({
  isAllowExternalRegistrations: settings.isAllowExternalRegistrations,
  isAllowMemberInvitations: settings.isAllowMemberInvitations,
  isMemberAllowedToCreateWorkspaces: settings.isMemberAllowedToCreateWorkspaces,
  isTwoFactorAuthRequired: settings.isTwoFactorAuthRequired,
});

export function SettingsComponent({ contentHeight }: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();
  const [settings, setSettings] = useState<InstanceSettings | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);

  // Scroll container ref for audit logs lazy loading
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Form state to track changes
  const [formSettings, setFormSettings] = useState<InstanceSettingsUpdate>({
    isAllowExternalRegistrations: false,
    isAllowMemberInvitations: false,
    isMemberAllowedToCreateWorkspaces: false,
    isTwoFactorAuthRequired: false,
  });

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setIsLoading(true);

    try {
      const currentSettings = await settingsApi.getSettings();
      setSettings(currentSettings);
      setFormSettings(toFormSettings(currentSettings));
      setHasChanges(false);
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsLoading(false);
    }
  };

  const handleSettingChange = (key: keyof InstanceSettingsUpdate, value: boolean) => {
    const newFormSettings = { ...formSettings, [key]: value };
    setFormSettings(newFormSettings);

    // Check if there are changes from the original settings
    if (settings) {
      const hasAnyChanges = Object.keys(newFormSettings).some(
        (settingKey) =>
          newFormSettings[settingKey as keyof InstanceSettingsUpdate] !==
          settings[settingKey as keyof InstanceSettingsUpdate],
      );
      setHasChanges(hasAnyChanges);
    }
  };

  const handleSave = async () => {
    if (!hasChanges) return;

    setIsSaving(true);
    try {
      const updatedSettings = await settingsApi.updateSettings(formSettings);
      setSettings(updatedSettings);
      setFormSettings(toFormSettings(updatedSettings));
      setHasChanges(false);
      message.success(t('settings.updated'));
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsSaving(false);
    }
  };

  const handleReset = () => {
    if (settings) {
      setFormSettings(toFormSettings(settings));
      setHasChanges(false);
    }
  };

  return (
    <div className="flex grow">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800"
          style={{ height: contentHeight }}
        >
          <h1 className="text-2xl font-bold dark:text-white">{t('settings.title')}</h1>

          <div className="mt-6">
            {isLoading ? (
              <div>
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            ) : (
              <div className="max-w-lg text-sm">
                <div className="space-y-6">
                  {/* External Registrations Setting */}
                  <div className="flex items-start justify-between border-b border-gray-200 pb-4 dark:border-gray-700">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900 dark:text-white">
                        {t('settings.externalRegistrations.title')}
                      </div>
                      <div className="mt-1 text-gray-500 dark:text-gray-400">
                        {t('settings.externalRegistrations.description')}
                      </div>
                    </div>

                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isAllowExternalRegistrations}
                        onChange={(checked) =>
                          handleSettingChange('isAllowExternalRegistrations', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isAllowExternalRegistrations
                            ? '#155dfc'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>

                  {/* Member Invitations Setting */}
                  {!formSettings.isAllowExternalRegistrations && (
                    <div className="flex items-start justify-between border-b border-gray-200 pb-4 dark:border-gray-700">
                      <div className="flex-1 pr-20">
                        <div className="font-medium text-gray-900 dark:text-white">
                          {t('settings.memberInvitations.title')}
                        </div>

                        <div className="mt-1 text-gray-500 dark:text-gray-400">
                          {t('settings.memberInvitations.description')}
                        </div>
                      </div>

                      <div className="ml-4">
                        <Switch
                          checked={formSettings.isAllowMemberInvitations}
                          onChange={(checked) =>
                            handleSettingChange('isAllowMemberInvitations', checked)
                          }
                          style={{
                            backgroundColor: formSettings.isAllowMemberInvitations
                              ? '#155dfc'
                              : undefined,
                          }}
                        />
                      </div>
                    </div>
                  )}

                  {/* Member Workspace Creation Setting */}
                  <div className="flex items-start justify-between border-b border-gray-200 pb-4 dark:border-gray-700">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900 dark:text-white">
                        {t('settings.memberWorkspaceCreation.title')}
                      </div>

                      <div className="mt-1 text-gray-500 dark:text-gray-400">
                        {t('settings.memberWorkspaceCreation.description')}
                      </div>
                    </div>
                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isMemberAllowedToCreateWorkspaces}
                        onChange={(checked) =>
                          handleSettingChange('isMemberAllowedToCreateWorkspaces', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isMemberAllowedToCreateWorkspaces
                            ? '#155dfc'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>

                  <div className="flex items-start justify-between border-b border-gray-200 pb-4 dark:border-gray-700">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900 dark:text-white">
                        {t('settings.twoFactorAuth.title')}
                      </div>

                      <div className="mt-1 text-gray-500 dark:text-gray-400">
                        {t('settings.twoFactorAuth.description')}
                      </div>

                      {!settings?.isEmailConfigured && (
                        <div className="mt-1 text-gray-500 dark:text-gray-400">
                          <Trans
                            i18nKey="settings.twoFactorAuth.mailServerRequired"
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
                      )}
                    </div>
                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isTwoFactorAuthRequired}
                        // Switching off needs no mail server, so an instance that has
                        // lost one can still be taken back to one factor from here.
                        disabled={
                          !settings?.isEmailConfigured && !settings?.isTwoFactorAuthRequired
                        }
                        onChange={(checked) =>
                          handleSettingChange('isTwoFactorAuthRequired', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isTwoFactorAuthRequired
                            ? '#155dfc'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>
                </div>

                {/* Action Buttons */}
                {hasChanges && (
                  <div className="mt-8 flex space-x-2">
                    <Button
                      type="primary"
                      onClick={handleSave}
                      loading={isSaving}
                      disabled={isSaving}
                      className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                    >
                      {isSaving ? t('settings.saving') : t('settings.saveChanges')}
                    </Button>

                    <Button type="default" onClick={handleReset} disabled={isSaving}>
                      {t('settings.reset')}
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            <Trans
              i18nKey="settings.readMore"
              components={{
                docsLink: (
                  <a
                    href={getWebsitePageUrl('accessManagementGlobalSettings', locale)}
                    target="_blank"
                    rel="noreferrer"
                    className="!text-blue-600"
                  />
                ),
              }}
            />
          </div>

          {/* Health-check Information */}
          <div className="my-8 max-w-2xl">
            <h2 className="mb-3 text-xl font-bold dark:text-white">
              {t('settings.healthcheck.title')}
            </h2>

            <div className="group relative">
              <div className="flex items-center rounded-md border border-gray-300 bg-gray-50 px-3 py-2 !font-mono text-sm text-gray-700 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200">
                <code
                  className="flex-1 cursor-pointer break-all transition-colors select-all hover:text-blue-600"
                  onClick={() => {
                    window.open(`${getApplicationServer()}/api/v1/system/health`, '_blank');
                  }}
                  title={t('settings.healthcheck.openInNewTab')}
                >
                  {getApplicationServer()}/api/v1/system/health
                </code>
                <Button
                  type="text"
                  size="small"
                  className="ml-2 opacity-0 transition-opacity group-hover:opacity-100"
                  onClick={() => {
                    ClipboardHelper.copyToClipboard(
                      `${getApplicationServer()}/api/v1/system/health`,
                    );
                    message.success(t('settings.healthcheck.copied'));
                  }}
                >
                  📋
                </Button>
              </div>
              <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {t('settings.healthcheck.description')}
              </div>
            </div>
          </div>

          <VerificationAgentsComponent />

          {settings && <MailServerComponent isEmailConfigured={settings.isEmailConfigured} />}

          <AuditLogsComponent scrollContainerRef={scrollContainerRef} />
        </div>
      </div>
    </div>
  );
}
