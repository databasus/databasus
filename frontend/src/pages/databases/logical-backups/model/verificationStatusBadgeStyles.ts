/* eslint-disable i18next/no-literal-string -- Tailwind class names */
import { LogicalBackupVerificationStatus } from '@/pages/databases/logical-backups/model/LogicalBackupVerificationStatus';

interface VerificationBadgeStyle {
  pillClasses: string;
  dotClasses: string;
}

// NOT_VERIFIED is intentionally absent: a lookup miss is how the UI decides to
// render no tag at all for backups that were never restore-verified.
export const VERIFICATION_STATUS_BADGE_STYLES: Partial<
  Record<LogicalBackupVerificationStatus, VerificationBadgeStyle>
> = {
  [LogicalBackupVerificationStatus.VERIFIED_SUCCESSFUL]: {
    pillClasses:
      'bg-green-500/10 text-green-700 ring-green-500/30 dark:bg-green-500/15 dark:text-green-300 dark:ring-green-400/30',
    dotClasses: 'bg-green-500 dark:bg-green-400',
  },
  [LogicalBackupVerificationStatus.VERIFICATION_FAILED]: {
    pillClasses:
      'bg-rose-500/10 text-rose-700 ring-rose-500/30 dark:bg-rose-500/15 dark:text-rose-300 dark:ring-rose-400/30',
    dotClasses: 'bg-rose-500 dark:bg-rose-400',
  },
};
