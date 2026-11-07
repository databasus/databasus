import { createContext } from 'react';

import type { FeedbackNotice } from '@/shared/lib/feedback/FeedbackNotice';

interface FeedbackContextValue {
  show: (notice: FeedbackNotice, owner?: string) => void;
  dismissOwner: (owner: string) => void;
}

export const FeedbackContext = createContext<FeedbackContextValue | undefined>(undefined);
