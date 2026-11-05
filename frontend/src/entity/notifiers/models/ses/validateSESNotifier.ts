import type { SESNotifier } from './SESNotifier';

export const validateSESNotifier = (notifier: SESNotifier): boolean => {
  if (!notifier.fromEmail?.trim()) {
    return false;
  }

  if (!notifier.targetEmail?.trim()) {
    return false;
  }

  if (!notifier.region?.trim()) {
    return false;
  }

  if (notifier.authType === 'KEYS') {
    if (!notifier.accessKey?.trim() || !notifier.secretKey?.trim()) {
      return false;
    }
  }

  return true;
};
