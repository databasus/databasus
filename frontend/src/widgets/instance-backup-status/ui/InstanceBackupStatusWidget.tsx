import { useEffect } from 'react';

import {
  clearRestoredSignInExpectation,
  useInstanceBackupStatus,
  useRefreshInstanceBackupStatus,
} from '@/entities/instance-backups';
import { selectFloatingNotice } from '@/widgets/instance-backup-status/model/selectFloatingNotice';
import { FloatingBackupNotice } from '@/widgets/instance-backup-status/ui/FloatingBackupNotice';
import { RecoveryPauseNotice } from '@/widgets/instance-backup-status/ui/RecoveryPauseNotice';

interface Props {
  isRootAdmin: boolean;
  isBackupSectionOpen: boolean;
  onOpenBackupSection: () => void;
}

// Cross-screen protection state for root admin: the recovery pause and one floating notice. Master-key
// onboarding replaces the whole application, so it is not part of this widget.
export function InstanceBackupStatusWidget({
  isRootAdmin,
  isBackupSectionOpen,
  onOpenBackupSection,
}: Props) {
  const statusQuery = useInstanceBackupStatus({ isEnabled: isRootAdmin });
  const refreshStatus = useRefreshInstanceBackupStatus();

  const status = statusQuery.data;
  const isStatusLoaded = !!status;

  useEffect(() => {
    if (isStatusLoaded) clearRestoredSignInExpectation();
  }, [isStatusLoaded]);

  if (!isRootAdmin || !status) return null;

  const notice = selectFloatingNotice(status);

  return (
    <>
      {status.isRecoveryPaused && (
        <RecoveryPauseNotice recovery={status.recovery} onActivated={refreshStatus} />
      )}
      {notice && !isBackupSectionOpen && (
        <FloatingBackupNotice
          notice={notice}
          status={status}
          isStatusStale={!!statusQuery.error}
          onOpenBackupSection={onOpenBackupSection}
          onRetryStatus={() => void statusQuery.refetch()}
        />
      )}
    </>
  );
}
