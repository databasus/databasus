import { getSessionSnapshot } from '@/shared/auth';

export const getSignedInUserId = (): string => {
  const userId = getSessionSnapshot().userId;
  if (!userId) {
    throw new Error('no signed-in user is stored');
  }

  return userId;
};
