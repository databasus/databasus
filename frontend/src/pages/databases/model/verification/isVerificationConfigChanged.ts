import { isScheduleChanged } from '../../../../entities/schedules';
import type { VerificationConfig } from './VerificationConfig';
import { VerificationScheduleType } from './VerificationScheduleType';

interface VerificationConfigComparison {
  acknowledged: VerificationConfig;
  shown: VerificationConfig;
}

const areTriggersChanged = ({ acknowledged, shown }: VerificationConfigComparison): boolean =>
  acknowledged.notificationTriggers.length !== shown.notificationTriggers.length ||
  shown.notificationTriggers.some(
    (trigger) => !acknowledged.notificationTriggers.includes(trigger),
  );

// Compares what the form shows, so an edit it no longer shows never counts as a difference.
export const isVerificationConfigChanged = ({
  acknowledged,
  shown,
}: VerificationConfigComparison): boolean => {
  if (acknowledged.isAutomaticVerificationEnabled !== shown.isAutomaticVerificationEnabled) {
    return true;
  }

  if (!shown.isAutomaticVerificationEnabled) return false;

  if (
    acknowledged.scheduleType !== shown.scheduleType ||
    areTriggersChanged({ acknowledged, shown })
  ) {
    return true;
  }

  return (
    shown.scheduleType === VerificationScheduleType.INTERVAL &&
    isScheduleChanged({ acknowledged: acknowledged.schedule, shown: shown.schedule })
  );
};
