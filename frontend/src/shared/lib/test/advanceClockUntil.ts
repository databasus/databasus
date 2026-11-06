import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';

const POLL_INTERVAL_MS = 1_000;

// Moves the clock one poll at a time and yields between the steps, because a poll waits for the
// answer of the poll before it.
export const advanceClockUntil = (condition: () => void) =>
  waitForCondition(async () => {
    await advanceFakeClock(POLL_INTERVAL_MS);
    condition();
  });
