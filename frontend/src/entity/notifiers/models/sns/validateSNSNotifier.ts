import type { SNSNotifier } from './SNSNotifier';

export const validateSNSNotifier = (notifier: SNSNotifier): boolean => {
  if (!notifier.topicArn?.trim()) {
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
