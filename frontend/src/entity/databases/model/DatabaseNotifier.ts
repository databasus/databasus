import type { ApiComponents } from '../../../shared/api';

// A notifier a database uses, as the /databases responses show it. /databases is not in the
// contract yet, so this shape is written by hand; a request names the notifier by its ID alone.
export interface DatabaseNotifier {
  id: string;
  workspaceId: string;
  name: string;
  notifierType: ApiComponents['schemas']['NotifierSettings']['type'];
  lastSendError?: string | null;
}
