import { Button, CloseButton } from '@mantine/core';
import { KeyRound, ShieldAlert, ShieldCheck } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  INSTANCE_BACKUP_INCIDENT_KIND_LABEL_KEYS,
  type InstanceBackupStatus,
} from '@/entities/instance-backups';
import type { FloatingNotice } from '@/widgets/instance-backup-status/model/selectFloatingNotice';

interface Props {
  notice: FloatingNotice;
  status: InstanceBackupStatus;
  isStatusStale: boolean;
  isDeferring: boolean;
  onOpenBackupSection: () => void;
  onOpenMasterKey: () => void;
  onDeferSetup: () => void;
  onRetryStatus: () => void;
}

// On a narrow screen the notice sits in the page flow, so it never covers controls; on a wider one
// it floats in a corner. It is a status region and never takes focus.
export function FloatingBackupNotice({
  notice,
  status,
  isStatusStale,
  isDeferring,
  onOpenBackupSection,
  onOpenMasterKey,
  onDeferSetup,
  onRetryStatus,
}: Props) {
  const { t } = useTranslation();
  const isProblem = notice.kind === 'incident';
  const Icon =
    notice.kind === 'master-key'
      ? KeyRound
      : notice.kind === 'resuming'
        ? ShieldCheck
        : ShieldAlert;

  const title =
    notice.kind === 'incident' && status.incident
      ? t(INSTANCE_BACKUP_INCIDENT_KIND_LABEL_KEYS[status.incident.kind])
      : notice.kind === 'resuming'
        ? t('instanceBackupNotices.resuming')
        : notice.kind === 'master-key'
          ? t('instanceBackupNotices.masterKey')
          : t('instanceBackupNotices.setup');

  return (
    <section
      role="status"
      aria-label={t('instanceBackupNotices.label')}
      className={`mb-3 rounded border bg-raised p-3 text-sm shadow md:fixed md:right-4 md:bottom-4 md:z-[100] md:mb-0 md:w-96 md:max-w-[calc(100vw-2rem)] ${
        isProblem ? 'border-danger' : 'border-border'
      }`}
    >
      <div className="flex items-start gap-2">
        <Icon
          aria-hidden
          size={18}
          className={`mt-0.5 shrink-0 ${isProblem ? 'text-danger' : 'text-accent'}`}
        />
        <div className="min-w-0 flex-1">
          <p className="font-medium">{title}</p>
          {notice.kind === 'setup' && (
            <p className="mt-1 text-muted">{t('instanceBackupNotices.setupDescription')}</p>
          )}
          {isStatusStale && (
            <p className="mt-1 text-muted">
              {t('instanceBackupNotices.stale')}{' '}
              <Button size="compact-xs" variant="subtle" onClick={onRetryStatus}>
                {t('common.actions.retry')}
              </Button>
            </p>
          )}
          <div className="mt-2 flex flex-wrap gap-2">
            {notice.kind === 'master-key' ? (
              <Button size="xs" variant="filled" onClick={onOpenMasterKey}>
                {t('instanceBackupNotices.saveMasterKey')}
              </Button>
            ) : (
              <Button size="xs" variant="filled" onClick={onOpenBackupSection}>
                {notice.kind === 'setup'
                  ? t('instanceBackupNotices.configure')
                  : t('instanceBackupNotices.open')}
              </Button>
            )}
            {notice.kind === 'setup' && (
              <Button size="xs" variant="default" loading={isDeferring} onClick={onDeferSetup}>
                {t('instanceBackupNotices.remindTomorrow')}
              </Button>
            )}
          </div>
        </div>
        {notice.kind === 'setup' && (
          <CloseButton
            size="sm"
            aria-label={t('instanceBackupNotices.remindTomorrow')}
            disabled={isDeferring}
            onClick={onDeferSetup}
          />
        )}
      </div>
    </section>
  );
}
