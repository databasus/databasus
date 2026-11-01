import { isScheduleComplete } from '../../../../entities/schedules';
import type { VerificationConfig } from './VerificationConfig';
import { VerificationScheduleType } from './VerificationScheduleType';

// The server validates the schedule only while verification is on and runs on it.
export const isVerificationConfigComplete = (config: VerificationConfig): boolean => {
  if (!config.isAutomaticVerificationEnabled) return true;
  if (config.scheduleType === VerificationScheduleType.AFTER_BACKUP) return true;

  return isScheduleComplete(config.schedule);
};
