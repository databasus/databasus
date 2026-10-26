import type { ApiComponents } from '../../../../shared/api';

// A header of the webhook's requests. A saved header comes without its value, and a header sent
// without one keeps the value stored under its name.
export type WebhookHeader = ApiComponents['schemas']['WebhookHeader'];
