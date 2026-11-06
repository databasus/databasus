import { useSyncExternalStore } from 'react';

import { getSessionSnapshot, subscribeSession } from '@/shared/auth/session';

export const useSession = () => useSyncExternalStore(subscribeSession, getSessionSnapshot);
