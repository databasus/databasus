import { Button } from '@mantine/core';
import { ArrowRight, ShieldAlert, ShieldCheck, TriangleAlert } from 'lucide-react';
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
  onOpenBackupSection: () => void;
  onRetryStatus: () => void;
}

// On a narrow screen the notice sits in the page flow, so it never covers controls; on a wider one
// it floats in a corner. It is a status region and never takes focus.
export function FloatingBackupNotice({
  notice,
  status,
  isStatusStale,
  onOpenBackupSection,
  onRetryStatus,
}: Props) {
  const { t } = useTranslation();
  const Icon =
    notice.kind === 'resuming'
      ? ShieldCheck
      : notice.kind === 'setup'
        ? TriangleAlert
        : ShieldAlert;

  const title =
    notice.kind === 'incident' && status.incident
      ? t(INSTANCE_BACKUP_INCIDENT_KIND_LABEL_KEYS[status.incident.kind])
      : notice.kind === 'resuming'
        ? t('instanceBackupNotices.resuming')
        : t('instanceBackupNotices.setup');

  // Setup is a warning, an incident is a failure and resumption is progress.
  const isSetup = notice.kind === 'setup';
  const isIncident = notice.kind === 'incident';

  return (
    <section
      role="status"
      aria-label={t('instanceBackupNotices.label')}
      className={`mb-3 flex items-start gap-3 rounded-lg border p-4 text-sm shadow-lg md:fixed md:right-4 md:bottom-4 md:z-[100] md:mb-0 md:w-96 md:max-w-[calc(100vw-2rem)] ${
        isSetup
          ? 'border-warning-border bg-warning-soft'
          : isIncident
            ? 'border-danger-border bg-danger-soft'
            : 'border-border bg-raised'
      }`}
    >
      <Icon
        aria-hidden
        size={20}
        className={`mt-0.5 shrink-0 ${
          isSetup ? 'text-warning-fill' : isIncident ? 'text-danger' : 'text-accent'
        }`}
      />
      <div className="min-w-0 flex-1">
        <p className="font-semibold">{title}</p>
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
        <Button
          className="mt-3"
          size="xs"
          variant="filled"
          color={isIncident ? 'danger' : undefined}
          rightSection={<ArrowRight aria-hidden size={14} />}
          onClick={onOpenBackupSection}
        >
          {notice.kind === 'setup'
            ? t('instanceBackupNotices.configure')
            : t('instanceBackupNotices.open')}
        </Button>
      </div>
    </section>
  );
}
