import { Alert, Button, Loader } from '@mantine/core';
import { CircleAlert, Mail } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  INSTANCE_BACKUP_HEALTH_LABEL_KEYS,
  INSTANCE_BACKUP_INCIDENT_KIND_LABEL_KEYS,
  useInstanceBackupStatus,
  useRefreshInstanceBackupStatus,
} from '@/entities/instance-backups';
import type { Workspace } from '@/entities/workspaces';
import { InstanceBackupHistory } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupHistory';
import { InstanceBackupSettingsForm } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupSettingsForm';
import { MasterKeyPanel } from '@/pages/instance-settings/ui/instance-backups/MasterKeyPanel';
import { InstanceRecoveryImport } from '@/pages/instance-settings/ui/instance-backups/recovery/InstanceRecoveryImport';
import { useSession } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { formatInstant } from '@/shared/time';
import { LoadFailure } from '@/shared/ui';

export const INSTANCE_BACKUPS_HEADING_ID = 'databasus-backups';

interface Props {
  workspaces: Workspace[];
  onWorkspaceCreated: (workspace: Workspace) => void;
  onConfigureSmtp: () => void;
}

export function InstanceBackupsSection({ workspaces, onWorkspaceCreated, onConfigureSmtp }: Props) {
  const { t } = useTranslation();
  const statusQuery = useInstanceBackupStatus({ isEnabled: true });
  const refreshStatus = useRefreshInstanceBackupStatus();
  const session = useSession();
  const status = statusQuery.data;

  return (
    <section className="my-8 max-w-2xl text-sm" aria-labelledby={INSTANCE_BACKUPS_HEADING_ID}>
      <h2 id={INSTANCE_BACKUPS_HEADING_ID} tabIndex={-1} className="mb-3 text-xl font-bold">
        {t('instanceBackups.title')}
      </h2>
      <p className="text-muted">{t('instanceBackups.description')}</p>
      <p className="mt-1 text-muted">{t('instanceBackups.retainSeparately')}</p>

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
          {statusQuery.error && (
            <Alert
              className="mt-3"
              color="warning"
              icon={<CircleAlert aria-hidden size={16} />}
              title={t('instanceBackups.status.refreshFailed', {
                reason: translateApiError(statusQuery.error, t),
              })}
            />
          )}
          <div className="mt-3" role="status">
            <span className="font-medium">{t('instanceBackups.status.label')}</span>{' '}
            <span>{t(INSTANCE_BACKUP_HEALTH_LABEL_KEYS[status.health])}</span>
          </div>
          {status.incident && (
            <Alert
              className="mt-3"
              color="danger"
              icon={<CircleAlert aria-hidden size={16} />}
              title={t(INSTANCE_BACKUP_INCIDENT_KIND_LABEL_KEYS[status.incident.kind])}
            >
              <div>
                {t('instanceBackups.incidents.since', {
                  time: formatInstant(status.incident.startedAt),
                })}
              </div>
              {status.incident.lastMailError && (
                <div className="mt-1 break-words">
                  {t('instanceBackups.incidents.mailFailed', {
                    error: status.incident.lastMailError,
                  })}
                </div>
              )}
            </Alert>
          )}
          {!status.isEmailConfigured ? (
            <Alert
              className="mt-3"
              color="warning"
              icon={<Mail aria-hidden size={16} />}
              title={t('instanceBackups.smtp.missingTitle')}
            >
              <p>{t('instanceBackups.smtp.missingDescription')}</p>
              <Button className="mt-2" size="xs" variant="outline" onClick={onConfigureSmtp}>
                {t('instanceBackups.smtp.configure')}
              </Button>
            </Alert>
          ) : (
            !status.isRootEmailUsable && (
              <Alert
                className="mt-3"
                color="warning"
                icon={<Mail aria-hidden size={16} />}
                title={t('instanceBackups.smtp.addressMissing')}
              />
            )
          )}

          <MasterKeyPanel status={status} onAcknowledged={() => void refreshStatus()} />

          <InstanceBackupSettingsForm
            destinationHealth={status.destinations}
            workspaces={workspaces}
            onWorkspaceCreated={onWorkspaceCreated}
            onSaved={() => void refreshStatus()}
          />
          <InstanceBackupHistory
            hasDestinations={status.destinations.length > 0}
            onStarted={() => void refreshStatus()}
          />
          <InstanceRecoveryImport
            key={session.generation}
            identityProofMethods={status.identityProofMethods}
          />
        </>
      )}
    </section>
  );
}
