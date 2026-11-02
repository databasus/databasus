import { isWebhookHeaderValueMissing } from './isWebhookHeaderValueMissing';
import type { WebhookHeader } from './WebhookHeader';
import type { WebhookNotifierSettings } from './WebhookNotifierSettings';

export const validateWebhookNotifier = (
  settings: WebhookNotifierSettings,
  headersRequiringValue: WebhookHeader[] = [],
): boolean =>
  !!settings.webhookUrl &&
  !settings.headers.some((header) => isWebhookHeaderValueMissing(header, headersRequiringValue));
