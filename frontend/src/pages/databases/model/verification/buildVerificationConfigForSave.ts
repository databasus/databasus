import type { ApiComponents } from '../../../../shared/api';
import type { VerificationConfig } from './VerificationConfig';
import { VerificationScheduleType } from './VerificationScheduleType';

interface VerificationConfigForSaveSource {
  acknowledgedConfig: VerificationConfig;
  draft: VerificationConfig;
}

// Sends the acknowledged values for what the form hides. The server validates the schedule only while
// verification is on and runs on it, so a hidden invalid expression would otherwise be stored.
export const buildVerificationConfigForSave = ({
  acknowledgedConfig,
  draft,
}: VerificationConfigForSaveSource): ApiComponents['schemas']['VerificationConfigInput'] => {
  if (!draft.isAutomaticVerificationEnabled) {
    return {
      isAutomaticVerificationEnabled: false,
      scheduleType: acknowledgedConfig.scheduleType,
      schedule: acknowledgedConfig.schedule,
      notificationTriggers: acknowledgedConfig.notificationTriggers,
    };
  }

  return {
    isAutomaticVerificationEnabled: true,
    scheduleType: draft.scheduleType,
    schedule:
      draft.scheduleType === VerificationScheduleType.AFTER_BACKUP
        ? acknowledgedConfig.schedule
        : draft.schedule,
    notificationTriggers: draft.notificationTriggers,
  };
};
