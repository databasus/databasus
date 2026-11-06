import type { ApiComponents } from '@/shared/api';

// A header of the webhook's requests. A saved header comes without its value, and a header sent
// without one keeps the value stored under its name while the webhook URL stays the stored one.
export type WebhookHeader = ApiComponents['schemas']['WebhookHeader'];
