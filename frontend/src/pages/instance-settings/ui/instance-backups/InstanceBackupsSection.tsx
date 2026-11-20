import { Alert, Button, Loader } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { ArchiveRestore, CircleAlert, KeyRound, Settings2, TriangleAlert } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  InstanceBackupHealth,
  type InstanceBackupSettings,
  InstanceRecoveryStatus,
  instanceBackupApi,
  useInstanceBackupStatus,
  useRefreshInstanceBackupStatus,
} from '@/entities/instance-backups';
import type { Workspace } from '@/entities/workspaces';
import { InstanceBackupHistory } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupHistory';
import { InstanceBackupMailNotice } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupMailNotice';
import { InstanceBackupSettingsForm } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupSettingsForm';
import { InstanceBackupSettingsView } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupSettingsView';
import { InstanceBackupStatusBlock } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupStatusBlock';
import { MasterKeyDialog } from '@/pages/instance-settings/ui/instance-backups/MasterKeyDialog';
import { InstanceRecoveryDialog } from '@/pages/instance-settings/ui/instance-backups/recovery/InstanceRecoveryDialog';
import { useSessionQuery } from '@/shared/api';
import { useSession } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { Dialog, LoadFailure } from '@/shared/ui';

export const INSTANCE_BACKUPS_HEADING_ID = 'databasus-backups';

interface Props {
  workspaces: Workspace[];
  onWorkspaceCreated: (workspace: Workspace) => void;
  onConfigureSmtp: () => void;
}

export function InstanceBackupsSection({ workspaces, onWorkspaceCreated, onConfigureSmtp }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const session = useSession();
  const statusQuery = useInstanceBackupStatus({ isEnabled: true });
  const refreshStatus = useRefreshInstanceBackupStatus();
  const settingsQuery = useSessionQuery({
    queryKey: ['instance-backup-settings'],
    queryFn: (signal) => instanceBackupApi.getSettings({ signal }),
  });
  const [isEditingSettings, setIsEditingSettings] = useState(false);
  const [isMasterKeyOpen, setIsMasterKeyOpen] = useState(false);
  const [isRecoveryOpen, setIsRecoveryOpen] = useState(false);
  const status = statusQuery.data;
  const settings = settingsQuery.data;

  // A destination saved a moment ago counts before the next status read reports it.
  const isNotConfigured =
    status?.health === InstanceBackupHealth.NOT_CONFIGURED && !settings?.storageIds.length;

  const acceptSavedSettings = async (savedSettings: InstanceBackupSettings) => {
    queryClient.setQueryData(settingsQuery.queryKey, savedSettings);
    await refreshStatus();
  };

  const masterKeyButton = (
    <Button
      size="xs"
      variant="default"
      leftSection={<KeyRound aria-hidden size={14} />}
      onClick={() => setIsMasterKeyOpen(true)}
    >
      {t('masterKey.open')}
    </Button>
  );
  const recoveryButton = (
    <Button
      size="xs"
      variant="default"
      leftSection={<ArchiveRestore aria-hidden size={14} />}
      onClick={() => setIsRecoveryOpen(true)}
    >
      {t('instanceRecovery.open')}
    </Button>
  );

  const renderSettingsView = () => {
    if (!status) return null;

    if (settingsQuery.error) {
      return (
        <LoadFailure
          className="mt-6"
          error={settingsQuery.error}
          onRetry={() => void settingsQuery.refetch()}
        />
      );
    }

    if (!settings) return <Loader className="mt-6" role="status" />;

    return (
      <InstanceBackupSettingsView
        settings={settings}
        destinationHealth={status.destinations}
        onEdit={() => setIsEditingSettings(true)}
      />
    );
  };

  return (
    <section className="my-8 max-w-5xl text-sm" aria-labelledby={INSTANCE_BACKUPS_HEADING_ID}>
      <h2 id={INSTANCE_BACKUPS_HEADING_ID} tabIndex={-1} className="text-xl font-bold">
        {t('instanceBackups.title')}
      </h2>
      <p className="mt-1 max-w-xl text-muted">{t('instanceBackups.description')}</p>

      {statusQuery.error && !status ? (
        <LoadFailure
          className="mt-3"
          error={statusQuery.error}
          onRetry={() => void statusQuery.refetch()}
        />
      ) : !status ? (
        <Loader className="mt-3" role="status" />
      ) : (
        <>
          <div className="max-w-2xl">
            {statusQuery.error ? (
              <Alert
                className="mt-3"
                color="warning"
                icon={<CircleAlert aria-hidden size={16} />}
                title={t('instanceBackups.status.refreshFailed', {
                  reason: translateApiError(statusQuery.error, t),
                })}
              />
            ) : null}

            {/* The recent backups already show that a healthy instance is backed up. */}
            {status.health !== InstanceBackupHealth.HEALTHY && (
              <InstanceBackupStatusBlock status={status}>
                {isNotConfigured && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Button
                      size="xs"
                      variant="filled"
                      leftSection={<Settings2 aria-hidden size={14} />}
                      onClick={() => setIsEditingSettings(true)}
                    >
                      {t('instanceBackups.setup.start')}
                    </Button>
                    {recoveryButton}
                    {masterKeyButton}
                  </div>
                )}
              </InstanceBackupStatusBlock>
            )}
            <InstanceBackupMailNotice status={status} onConfigureSmtp={onConfigureSmtp} />
            {status.recovery?.status === InstanceRecoveryStatus.COMPLETED &&
              status.recovery.discardedCleanupCount > 0 && (
                <p className="mt-3 flex items-start gap-1.5 text-sm text-muted">
                  <TriangleAlert aria-hidden size={14} className="mt-0.5 shrink-0 text-warning" />
                  {t('instanceRecovery.discardedCleanup', {
                    count: status.recovery.discardedCleanupCount,
                  })}
                </p>
              )}
          </div>

          {!isNotConfigured && (
            <div className="grid grid-cols-1 gap-x-8 lg:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
              <div className="min-w-0">{renderSettingsView()}</div>
              <div className="min-w-0">
                <InstanceBackupHistory
                  hasDestinations={status.destinations.length > 0}
                  onStarted={() => void refreshStatus()}
                />
              </div>
            </div>
          )}

          <Dialog
            opened={isEditingSettings}
            onClose={() => setIsEditingSettings(false)}
            title={t('instanceBackups.settings.title')}
            size={560}
            closeOnClickOutside={false}
          >
            {isEditingSettings && (
              <InstanceBackupSettingsForm
                destinationHealth={status.destinations}
                workspaces={workspaces}
                onWorkspaceCreated={onWorkspaceCreated}
                onSaved={acceptSavedSettings}
                onClose={() => setIsEditingSettings(false)}
              />
            )}
          </Dialog>

          {!isNotConfigured && (
            <div className="mt-6 flex flex-wrap gap-2">
              {masterKeyButton}
              {recoveryButton}
            </div>
          )}

          {isMasterKeyOpen && (
            <MasterKeyDialog status={status} onClose={() => setIsMasterKeyOpen(false)} />
          )}
          {isRecoveryOpen && (
            <InstanceRecoveryDialog
              key={session.generation}
              isCodeRequired={status.isConfirmationCodeRequired}
              onClose={() => setIsRecoveryOpen(false)}
            />
          )}
        </>
      )}
    </section>
  );
}
