import { useSyncExternalStore } from 'react';

import { getSessionSnapshot, subscribeSession } from './session';

export const useSession = () => useSyncExternalStore(subscribeSession, getSessionSnapshot);
