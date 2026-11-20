import { useEffect } from 'react';

import {
  clearRestoredSignInExpectation,
  useInstanceBackupStatus,
} from '@/entities/instance-backups';
import { selectFloatingNotice } from '@/widgets/instance-backup-status/model/selectFloatingNotice';
import { FloatingBackupNotice } from '@/widgets/instance-backup-status/ui/FloatingBackupNotice';

interface Props {
  isRootAdmin: boolean;
  isBackupSectionOpen: boolean;
  onOpenBackupSection: () => void;
}

// Cross-screen protection state for root admin: one floating notice. Master-key onboarding replaces
// the whole application, so it is not part of this widget.
export function InstanceBackupStatusWidget({
  isRootAdmin,
  isBackupSectionOpen,
  onOpenBackupSection,
}: Props) {
  const statusQuery = useInstanceBackupStatus({ isEnabled: isRootAdmin });

  const status = statusQuery.data;
  const isStatusLoaded = !!status;

  useEffect(() => {
    if (isStatusLoaded) clearRestoredSignInExpectation();
  }, [isStatusLoaded]);

  if (!isRootAdmin || !status) return null;

  const notice = selectFloatingNotice(status);

  return (
    <>
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
