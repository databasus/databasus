import { isWebhookHeaderNameValid } from '@/entities/notifiers/model/webhook/isWebhookHeaderNameValid';
import { isWebhookHeaderValueMissing } from '@/entities/notifiers/model/webhook/isWebhookHeaderValueMissing';
import type { WebhookHeader } from '@/entities/notifiers/model/webhook/WebhookHeader';
import type { WebhookNotifierSettings } from '@/entities/notifiers/model/webhook/WebhookNotifierSettings';

export const validateWebhookNotifier = (
  settings: WebhookNotifierSettings,
  headersRequiringValue: WebhookHeader[] = [],
): boolean =>
  !!settings.webhookUrl &&
  settings.headers.every((header) => isWebhookHeaderNameValid(header.name)) &&
  !settings.headers.some((header) => isWebhookHeaderValueMissing(header, headersRequiringValue));
