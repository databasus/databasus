import { CopyOutlined, LoadingOutlined } from '@ant-design/icons';
import { Alert, App, Button, Spin, Switch } from 'antd';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type InstanceSettings, type InstanceSettingsUpdate, settingsApi } from '@/entities/users';
import { InstanceAuditLogs } from '@/pages/instance-settings/ui/InstanceAuditLogs';
import { MailServer } from '@/pages/instance-settings/ui/MailServer';
import { useSessionGuard } from '@/shared/auth';
import { getApplicationServer } from '@/shared/config';
import { useRequestRevision } from '@/shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { LoadFailure } from '@/shared/ui';

import { VerificationAgentsComponent } from './verification-agents/VerificationAgentsComponent';

interface Props {
  onSettingsChanged?: (settings: InstanceSettings) => void;
}

// The response carries the instance's answer about its mail server, which the
// form neither edits nor sends back.
const toFormSettings = (settings: InstanceSettings): InstanceSettingsUpdate => ({
  isAllowExternalRegistrations: settings.isAllowExternalRegistrations,
  isAllowMemberInvitations: settings.isAllowMemberInvitations,
  isMemberAllowedToCreateWorkspaces: settings.isMemberAllowedToCreateWorkspaces,
  isTwoFactorAuthRequired: settings.isTwoFactorAuthRequired,
});

export function InstanceSettingsPage({ onSettingsChanged }: Props) {
  const [settings, setSettings] = useState<InstanceSettings | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [failure, setFailure] = useState<unknown>();

  const [formSettings, setFormSettings] = useState<InstanceSettingsUpdate>({
    isAllowExternalRegistrations: false,
    isAllowMemberInvitations: false,
    isMemberAllowedToCreateWorkspaces: false,
    isTwoFactorAuthRequired: false,
  });

  const loadSettings = async () => {
    const revision = loadRevision.Invalidate();
    setIsLoading(true);
    setFailure(undefined);

    try {
      const currentSettings = await settingsApi.getSettings();
      if (!isCurrent() || !loadRevision.IsCurrent(revision)) return;
      setSettings(currentSettings);
      setFormSettings(toFormSettings(currentSettings));
    } catch (error: unknown) {
      if (isCurrent() && loadRevision.IsCurrent(revision)) setFailure(error);
    } finally {
      if (isCurrent() && loadRevision.IsCurrent(revision)) setIsLoading(false);
    }
  };

  const handleSettingChange = (key: keyof InstanceSettingsUpdate, value: boolean) => {
    const newFormSettings = { ...formSettings, [key]: value };
    setFormSettings(newFormSettings);
  };

  const handleSave = async () => {
    if (!hasChanges || isSaving) return;
    setFailure(undefined);
    const submitted = { ...formSettings };

    setIsSaving(true);
    try {
      const updatedSettings = await settingsApi.updateSettings(submitted);
      if (!isCurrent()) return;
      setSettings(updatedSettings);
      setFormSettings((current) => {
        const retained = { ...current };
        for (const key of Object.keys(submitted) as (keyof InstanceSettingsUpdate)[]) {
          if (current[key] === submitted[key]) retained[key] = updatedSettings[key];
        }
        return retained;
      });

      onSettingsChanged?.(updatedSettings);
      message.success(i18n.t('settings.updated'));
    } catch (error: unknown) {
      if (isCurrent()) setFailure(error);
    } finally {
      if (isCurrent()) setIsSaving(false);
    }
  };

  const handleReset = () => {
    if (settings) {
      setFormSettings(toFormSettings(settings));
    }
  };
  const { t, i18n } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();
  const isCurrent = useSessionGuard();
  const loadRevision = useRequestRevision();

  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const startSettingsLoad = useEffectEvent(() => {
    void loadSettings();
  });

  useEffect(() => {
    startSettingsLoad();
    return () => {
      loadRevision.Invalidate();
    };
  }, [loadRevision]);
  const hasChanges =
    !!settings &&
    (Object.keys(formSettings) as (keyof InstanceSettingsUpdate)[]).some(
      (key) => formSettings[key] !== settings[key],
    );

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <div className="flex min-h-0 w-full min-w-0 flex-1 flex-col">
        <div
          ref={scrollContainerRef}
          className="min-h-0 min-w-0 flex-1 overflow-y-auto rounded bg-surface p-4 wrap-anywhere shadow sm:p-5"
        >
          <h1 className="text-2xl font-bold">{t('settings.title')}</h1>

          {failure !== undefined &&
            (settings ? (
              <Alert
                className="mt-3"
                type="error"
                showIcon
                message={translateApiError(failure, t)}
              />
            ) : (
              <LoadFailure className="mt-3" error={failure} onRetry={() => void loadSettings()} />
            ))}
          <div className="mt-6">
            {isLoading ? (
              <div>
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            ) : settings ? (
              <div className="max-w-lg text-sm">
                <div className="space-y-6">
                  <div className="flex items-start justify-between gap-4 border-b border-border pb-4">
                    <div className="min-w-0 flex-1">
                      <div className="font-medium text-content">
                        {t('settings.externalRegistrations.title')}
                      </div>
                      <div className="mt-1 text-muted">
                        {t('settings.externalRegistrations.description')}
                      </div>
                    </div>

                    <div className="shrink-0">
                      <Switch
                        aria-label={t('settings.externalRegistrations.title')}
                        checked={formSettings.isAllowExternalRegistrations}
                        onChange={(checked) =>
                          handleSettingChange('isAllowExternalRegistrations', checked)
                        }
                      />
                    </div>
                  </div>
                  {!formSettings.isAllowExternalRegistrations && (
                    <div className="flex items-start justify-between gap-4 border-b border-border pb-4">
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-content">
                          {t('settings.memberInvitations.title')}
                        </div>

                        <div className="mt-1 text-muted">
                          {t('settings.memberInvitations.description')}
                        </div>
                      </div>

                      <div className="shrink-0">
                        <Switch
                          aria-label={t('settings.memberInvitations.title')}
                          checked={formSettings.isAllowMemberInvitations}
                          onChange={(checked) =>
                            handleSettingChange('isAllowMemberInvitations', checked)
                          }
                        />
                      </div>
                    </div>
                  )}
                  <div className="flex items-start justify-between gap-4 border-b border-border pb-4">
                    <div className="min-w-0 flex-1">
                      <div className="font-medium text-content">
                        {t('settings.memberWorkspaceCreation.title')}
                      </div>

                      <div className="mt-1 text-muted">
                        {t('settings.memberWorkspaceCreation.description')}
                      </div>
                    </div>
                    <div className="shrink-0">
                      <Switch
                        aria-label={t('settings.memberWorkspaceCreation.title')}
                        checked={formSettings.isMemberAllowedToCreateWorkspaces}
                        onChange={(checked) =>
                          handleSettingChange('isMemberAllowedToCreateWorkspaces', checked)
                        }
                      />
                    </div>
                  </div>

                  <div className="flex items-start justify-between gap-4 border-b border-border pb-4">
                    <div className="min-w-0 flex-1">
                      <div className="font-medium text-content">
                        {t('settings.twoFactorAuth.title')}
                      </div>

                      <div className="mt-1 text-muted">
                        {t('settings.twoFactorAuth.description')}
                      </div>

                      {!settings?.isEmailConfigured && (
                        <div className="mt-1 text-muted">
                          <Trans
                            i18nKey="settings.twoFactorAuth.mailServerRequired"
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
                      )}
                    </div>
                    <div className="shrink-0">
                      <Switch
                        aria-label={t('settings.twoFactorAuth.title')}
                        checked={formSettings.isTwoFactorAuthRequired}
                        // Switching off needs no mail server, so an instance that has
                        // lost one can still be taken back to one factor from here.
                        disabled={
                          !settings?.isEmailConfigured && !settings?.isTwoFactorAuthRequired
                        }
                        onChange={(checked) =>
                          handleSettingChange('isTwoFactorAuthRequired', checked)
                        }
                      />
                    </div>
                  </div>
                </div>
                {hasChanges && (
                  <div className="mt-8 flex flex-wrap gap-2">
                    <Button
                      className="!h-auto min-h-8 max-w-full !whitespace-normal"
                      type="primary"
                      aria-label={t('settings.saveChanges')}
                      onClick={handleSave}
                      loading={isSaving}
                      disabled={isSaving}
                    >
                      {isSaving ? t('settings.saving') : t('settings.saveChanges')}
                    </Button>

                    <Button
                      className="!h-auto min-h-8 max-w-full !whitespace-normal"
                      type="default"
                      onClick={handleReset}
                      disabled={isSaving}
                    >
                      {t('settings.reset')}
                    </Button>
                  </div>
                )}
              </div>
            ) : null}
          </div>

          <div className="mt-3 text-sm text-muted">
            <Trans
              i18nKey="settings.readMore"
              components={{
                docsLink: (
                  <a
                    href={getWebsitePageUrl('accessManagementGlobalSettings', locale)}
                    target="_blank"
                    rel="noreferrer"
                    className="!text-accent"
                  />
                ),
              }}
            />
          </div>
          <div className="my-8 max-w-2xl">
            <h2 className="mb-3 text-xl font-bold">{t('settings.healthcheck.title')}</h2>

            <div className="group relative">
              <div className="flex items-center rounded-md border border-border bg-surface px-3 py-2 !font-mono text-sm text-content">
                <a
                  href={`${getApplicationServer()}/api/v1/system/health`}
                  target="_blank"
                  rel="noreferrer"
                  className="flex-1 cursor-pointer break-all transition-colors select-all hover:text-accent"
                  title={t('settings.healthcheck.openInNewTab')}
                >
                  {getApplicationServer()}/api/v1/system/health
                </a>
                <Button
                  type="text"
                  size="small"
                  className="ml-2"
                  aria-label={t('common.actions.copy')}
                  icon={<CopyOutlined aria-hidden />}
                  onClick={() => {
                    BrowserClipboard.copyToClipboard(
                      `${getApplicationServer()}/api/v1/system/health`,
                    );
                    message.success(i18n.t('settings.healthcheck.copied'));
                  }}
                />
              </div>
              <div className="mt-1 text-xs text-muted">{t('settings.healthcheck.description')}</div>
            </div>
          </div>

          <VerificationAgentsComponent />

          {settings && <MailServer isEmailConfigured={settings.isEmailConfigured} />}

          <InstanceAuditLogs scrollContainerRef={scrollContainerRef} />
        </div>
      </div>
    </div>
  );
}
