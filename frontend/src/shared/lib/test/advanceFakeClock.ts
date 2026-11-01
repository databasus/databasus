import { act } from '@testing-library/react';
import { vi } from 'vitest';

export const advanceFakeClock = (elapsedMs: number) =>
  act(async () => {
    await vi.advanceTimersByTimeAsync(elapsedMs);
  });
