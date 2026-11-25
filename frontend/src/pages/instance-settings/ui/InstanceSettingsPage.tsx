import { ActionIcon, Alert, Button, Loader, Radio, Switch } from '@mantine/core';
import { CircleAlert, Copy } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  type InstanceSettings,
  type InstanceSettingsUpdate,
  TwoFactorRequirement,
  settingsApi,
} from '@/entities/users';
import type { Workspace } from '@/entities/workspaces';
import { focusSettingsHeading } from '@/pages/instance-settings/lib/focusSettingsHeading';
import { InstanceBackupsSection } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupsSection';
import { InstanceAuditLogs } from '@/pages/instance-settings/ui/InstanceAuditLogs';
import { MAIL_SERVER_HEADING_ID, MailServer } from '@/pages/instance-settings/ui/MailServer';
import { VerificationAgentsComponent } from '@/pages/instance-settings/ui/verification-agents/VerificationAgentsComponent';
import { useSessionMutation } from '@/shared/api';
import { useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { getApplicationServer } from '@/shared/config';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { useFeedback } from '@/shared/lib/feedback';
import { LoadFailure } from '@/shared/ui';

interface Props {
  isRootAdmin: boolean;
  workspaces?: Workspace[];
  onWorkspaceCreated?: (workspace: Workspace) => void;
  onSettingsChanged?: (settings: InstanceSettings) => void;
}

// The response carries the instance's answer about its mail server, which the
// form neither edits nor sends back.
const toFormSettings = (settings: InstanceSettings): InstanceSettingsUpdate => ({
  isAllowExternalRegistrations: settings.isAllowExternalRegistrations,
  isAllowMemberInvitations: settings.isAllowMemberInvitations,
  isMemberAllowedToCreateWorkspaces: settings.isMemberAllowedToCreateWorkspaces,
  twoFactorRequirement: settings.twoFactorRequirement,
});

const getPreselectedTwoFactorRequirement = (settings: InstanceSettings): TwoFactorRequirement =>
  settings.isEmailConfigured ? TwoFactorRequirement.EMAIL : TwoFactorRequirement.AUTHENTICATOR_APP;

export function InstanceSettingsPage({
  isRootAdmin,
  workspaces = [],
  onWorkspaceCreated,
  onSettingsChanged,
}: Props) {
  const updateSettingsMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof settingsApi.updateSettings>) =>
      settingsApi.updateSettings(...parameters),
  });

  const [settings, setSettings] = useState<InstanceSettings | undefined>(undefined);
  const query = useSessionQuery({
    queryKey: ['instance-settings-editor'],
    queryFn: (signal) => settingsApi.getSettings({ signal }),
  });
  const isLoading = query.isPending;
  const isSaving = updateSettingsMutation.isPending;
  const [actionFailure, setFailure] = useState<unknown>();
  const failure = actionFailure ?? query.error ?? undefined;

  const [formSettings, setFormSettings] = useState<InstanceSettingsUpdate>({
    isAllowExternalRegistrations: false,
    isAllowMemberInvitations: false,
    isMemberAllowedToCreateWorkspaces: false,
    twoFactorRequirement: TwoFactorRequirement.OFF,
  });

  const loadSettings = () => {
    setFailure(undefined);
    return query.refetch();
  };
  useEffect(() => {
    if (!query.data) return;
    setSettings(query.data);
    setFormSettings(toFormSettings(query.data));
  }, [query.data]);

  const handleSettingChange = <Key extends keyof InstanceSettingsUpdate>(
    key: Key,
    value: InstanceSettingsUpdate[Key],
  ) => {
    const newFormSettings = { ...formSettings, [key]: value };
    setFormSettings(newFormSettings);
  };

  const handleSave = async () => {
    if (!hasChanges || isSaving || isEmailMethodUnsavable) return;
    setFailure(undefined);
    const submitted = { ...formSettings };

    try {
      const updatedSettings = await updateSettingsMutation.mutateAsync([submitted]);
      if (!isCurrent()) return;
      setSettings(updatedSettings);
      setFormSettings((current) => {
        const retained: InstanceSettingsUpdate = { ...current };
        for (const key of Object.keys(submitted) as (keyof InstanceSettingsUpdate)[]) {
          if (current[key] === submitted[key])
            Object.assign(retained, { [key]: updatedSettings[key] });
        }
        return retained;
      });

      onSettingsChanged?.(updatedSettings);
      notify({ kind: 'success', title: i18n.t('settings.updated') });
    } catch (error: unknown) {
      if (isCurrent()) setFailure(error);
    } finally {
      if (isCurrent()) updateSettingsMutation.reset();
    }
  };

  const handleReset = () => {
    if (settings) {
      setFormSettings(toFormSettings(settings));
    }
  };
  const { t, i18n } = useTranslation();
  const { locale } = useLocale();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();

  const isSecondFactorRequired = formSettings.twoFactorRequirement !== TwoFactorRequirement.OFF;
  const isEmailMethodWithoutMailServer =
    !!settings &&
    !settings.isEmailConfigured &&
    formSettings.twoFactorRequirement === TwoFactorRequirement.EMAIL;
  // The instance refuses only a switch to the emailed code, so an email requirement saved before
  // the mail server was gone does not block saving other settings.
  const isEmailMethodUnsavable =
    isEmailMethodWithoutMailServer && settings?.twoFactorRequirement !== TwoFactorRequirement.EMAIL;

  const hasChanges =
    !!settings &&
    (Object.keys(formSettings) as (keyof InstanceSettingsUpdate)[]).some(
      (key) => formSettings[key] !== settings[key],
    );

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <div className="flex min-h-0 w-full min-w-0 flex-1 flex-col">
        <div className="min-h-0 min-w-0 flex-1 overflow-y-auto rounded bg-surface p-4 pb-100 wrap-anywhere shadow sm:p-5 sm:pb-100">
          <h1 className="text-2xl font-bold">{t('settings.title')}</h1>

          {failure !== undefined &&
            (settings ? (
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
              <LoadFailure className="mt-3" error={failure} onRetry={() => void loadSettings()} />
            ))}
          <div className="mt-6">
            {isLoading ? (
              <div>
                <Loader role="status" />
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
                        onChange={(event) =>
                          handleSettingChange(
                            'isAllowExternalRegistrations',
                            event.currentTarget.checked,
                          )
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
                          onChange={(event) =>
                            handleSettingChange(
                              'isAllowMemberInvitations',
                              event.currentTarget.checked,
                            )
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
                        onChange={(event) =>
                          handleSettingChange(
                            'isMemberAllowedToCreateWorkspaces',
                            event.currentTarget.checked,
                          )
                        }
                      />
                    </div>
                  </div>

                  <div className="border-b border-border pb-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-content">
                          {t('settings.twoFactorAuth.title')}
                        </div>

                        <div className="mt-1 text-muted">
                          {t('settings.twoFactorAuth.description')}
                        </div>
                      </div>
                      <div className="shrink-0">
                        <Switch
                          aria-label={t('settings.twoFactorAuth.title')}
                          checked={isSecondFactorRequired}
                          onChange={(event) =>
                            handleSettingChange(
                              'twoFactorRequirement',
                              event.currentTarget.checked
                                ? getPreselectedTwoFactorRequirement(settings)
                                : TwoFactorRequirement.OFF,
                            )
                          }
                        />
                      </div>
                    </div>

                    {isSecondFactorRequired && (
                      <Radio.Group
                        className="mt-3"
                        label={t('settings.twoFactorAuth.method')}
                        value={formSettings.twoFactorRequirement}
                        onChange={(value) =>
                          handleSettingChange(
                            'twoFactorRequirement',
                            value === TwoFactorRequirement.EMAIL
                              ? TwoFactorRequirement.EMAIL
                              : TwoFactorRequirement.AUTHENTICATOR_APP,
                          )
                        }
                      >
                        <div className="mt-2 space-y-2">
                          <Radio
                            value={TwoFactorRequirement.EMAIL}
                            label={t('settings.twoFactorAuth.methods.email')}
                            description={
                              isEmailMethodWithoutMailServer && (
                                <span className="text-warning">
                                  <Trans
                                    i18nKey="settings.twoFactorAuth.mailServerRequired"
                                    components={{
                                      mailServerLink: (
                                        <button
                                          type="button"
                                          className="cursor-pointer text-accent hover:underline"
                                          onClick={() =>
                                            focusSettingsHeading(MAIL_SERVER_HEADING_ID)
                                          }
                                        />
                                      ),
                                    }}
                                  />
                                </span>
                              )
                            }
                          />
                          <Radio
                            value={TwoFactorRequirement.AUTHENTICATOR_APP}
                            label={t('settings.twoFactorAuth.methods.authenticatorApp')}
                            description={t('settings.twoFactorAuth.authenticatorAppExamples')}
                          />
                        </div>
                      </Radio.Group>
                    )}
                  </div>
                </div>
                {hasChanges && (
                  <div className="mt-8 flex flex-wrap gap-2">
                    <Button
                      className="!h-auto min-h-8 max-w-full !whitespace-normal"
                      variant="filled"
                      aria-label={t('settings.saveChanges')}
                      onClick={handleSave}
                      loading={isSaving}
                      disabled={isSaving || isEmailMethodUnsavable}
                      aria-busy={isSaving}
                    >
                      {isSaving ? t('settings.saving') : t('settings.saveChanges')}
                    </Button>

                    <Button
                      className="!h-auto min-h-8 max-w-full !whitespace-normal"
                      variant="default"
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
                <ActionIcon
                  variant="subtle"
                  size="xs"
                  className="ml-2"
                  aria-label={t('common.actions.copy')}
                  onClick={() => {
                    BrowserClipboard.copyToClipboard(
                      `${getApplicationServer()}/api/v1/system/health`,
                    );
                    notify({ kind: 'success', title: i18n.t('settings.healthcheck.copied') });
                  }}
                >
                  <Copy aria-hidden size="1em" className="inline-block shrink-0 align-middle" />
                </ActionIcon>
              </div>
              <div className="mt-1 text-xs text-muted">{t('settings.healthcheck.description')}</div>
            </div>
          </div>

          {isRootAdmin && (
            <InstanceBackupsSection
              workspaces={workspaces}
              onWorkspaceCreated={(workspace) => onWorkspaceCreated?.(workspace)}
              onConfigureSmtp={() => focusSettingsHeading(MAIL_SERVER_HEADING_ID)}
            />
          )}

          <VerificationAgentsComponent />

          {settings && <MailServer isEmailConfigured={settings.isEmailConfigured} />}

          <InstanceAuditLogs />
        </div>
      </div>
    </div>
  );
}
