import {
  MattermostDeliveryMode,
  type NotifierSettings,
  NotifierType,
  WebhookMethod,
  getDefaultEmailNotifierSecurity,
} from '@/entities/notifiers';
import { DEFAULT_ACCEPTED_CATEGORIES } from '@/features/edit-notifier/lib/defaultAcceptedCategories';

const UNSET_SMTP_PORT = 0;

// The empty settings a new notifier of the type starts from.
export const createDefaultNotifierSettings = (type: NotifierType): NotifierSettings => {
  switch (type) {
    case NotifierType.EMAIL:
      return {
        type,
        targetEmail: '',
        smtpHost: '',
        smtpPort: UNSET_SMTP_PORT,
        isInsecureSkipVerify: false,
        security: getDefaultEmailNotifierSecurity(UNSET_SMTP_PORT),
      };
    case NotifierType.TELEGRAM:
      return { type, targetChatId: '', isProxyEnabled: false };
    case NotifierType.SLACK:
      return { type, targetChatId: '' };
    case NotifierType.DISCORD:
      return { type };
    case NotifierType.TEAMS:
      return { type };
    case NotifierType.MATTERMOST:
      return {
        type,
        deliveryMode: MattermostDeliveryMode.WEBHOOK,
        isInsecureSkipVerify: false,
      };
    case NotifierType.WEBHOOK:
      return {
        type,
        webhookUrl: '',
        webhookMethod: WebhookMethod.POST,
        headers: [],
        acceptedCategories: [...DEFAULT_ACCEPTED_CATEGORIES],
      };
  }
};
