import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  clearRestoredSignInExpectation,
  instanceBackupApi,
  useInstanceBackupStatus,
  useRefreshInstanceBackupStatus,
} from '@/entities/instance-backups';
import { useSessionMutation } from '@/shared/api';
import { useSession, useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import {
  deferMasterKeyOnboarding,
  isMasterKeyOnboardingDeferred,
} from '@/widgets/instance-backup-status/model/masterKeyOnboardingDeferral';
import { selectFloatingNotice } from '@/widgets/instance-backup-status/model/selectFloatingNotice';
import { FloatingBackupNotice } from '@/widgets/instance-backup-status/ui/FloatingBackupNotice';
import { MasterKeyOnboardingDialog } from '@/widgets/instance-backup-status/ui/MasterKeyOnboardingDialog';
import { RecoveryPauseNotice } from '@/widgets/instance-backup-status/ui/RecoveryPauseNotice';

interface Props {
  isRootAdmin: boolean;
  isBackupSectionOpen: boolean;
  onOpenBackupSection: () => void;
}

// Cross-screen protection state for root admin: master-key onboarding and one floating notice.
// Deferring onboarding lasts for the current sign-in, reloads included, so a later sign-in shows
// the dialog again.
export function InstanceBackupStatusWidget({
  isRootAdmin,
  isBackupSectionOpen,
  onOpenBackupSection,
}: Props) {
  const { i18n } = useTranslation();
  const notify = useFeedback();
  const session = useSession();
  const isCurrent = useSessionGuard();
  const statusQuery = useInstanceBackupStatus({ isEnabled: isRootAdmin });
  const refreshStatus = useRefreshInstanceBackupStatus();
  // Deferral lives in sessionStorage; bumping this re-renders after it changes.
  const [, setDeferralRevision] = useState(0);
  const [isOnboardingReopened, setIsOnboardingReopened] = useState(false);
  const deferReminderMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof instanceBackupApi.updateReminder>) =>
      instanceBackupApi.updateReminder(...parameters),
  });

  const status = statusQuery.data;
  const isStatusLoaded = !!status;

  useEffect(() => {
    if (isStatusLoaded) clearRestoredSignInExpectation();
  }, [isStatusLoaded]);

  if (!isRootAdmin || !status) return null;

  const isOnboardingDeferred = isMasterKeyOnboardingDeferred(session) && !isOnboardingReopened;
  const isOnboardingOpen = !status.isMasterKeyAcknowledged && !isOnboardingDeferred;
  const notice = selectFloatingNotice({
    status,
    isMasterKeyOnboardingDeferred: isOnboardingDeferred,
    // The status read is polled, so its fetch time is the current time of this decision.
    now: new Date(statusQuery.dataUpdatedAt),
  });

  const deferSetup = async () => {
    if (deferReminderMutation.isPending) return;

    try {
      await deferReminderMutation.mutateAsync([true]);
      if (isCurrent()) await refreshStatus();
    } catch (error: unknown) {
      if (isCurrent()) notify({ kind: 'error', title: translateApiError(error, i18n.t) });
    } finally {
      if (isCurrent()) deferReminderMutation.reset();
    }
  };

  return (
    <>
      {status.isRecoveryPaused && (
        <RecoveryPauseNotice recovery={status.recovery} onActivated={refreshStatus} />
      )}
      {isOnboardingOpen && (
        <MasterKeyOnboardingDialog
          status={status}
          onAcknowledged={() => void refreshStatus()}
          onDefer={() => {
            deferMasterKeyOnboarding(session);
            setDeferralRevision((revision) => revision + 1);
            setIsOnboardingReopened(false);
          }}
        />
      )}
      {notice && !isBackupSectionOpen && !isOnboardingOpen && (
        <FloatingBackupNotice
          notice={notice}
          status={status}
          isStatusStale={!!statusQuery.error}
          isDeferring={deferReminderMutation.isPending}
          onOpenBackupSection={onOpenBackupSection}
          onOpenMasterKey={() => setIsOnboardingReopened(true)}
          onDeferSetup={() => void deferSetup()}
          onRetryStatus={() => void statusQuery.refetch()}
        />
      )}
    </>
  );
}
