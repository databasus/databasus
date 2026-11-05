import type { WebhookHeader } from './WebhookHeader';

// Header names are matched without regard to letter case, as the backend matches them.
export const isWebhookHeaderValueMissing = (
  header: WebhookHeader,
  headersRequiringValue: WebhookHeader[],
): boolean =>
  !header.value &&
  headersRequiringValue.some(
    (requiredHeader) => requiredHeader.name.toLowerCase() === header.name.toLowerCase(),
  );
