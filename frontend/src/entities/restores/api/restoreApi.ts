import type { Restore } from '@/entities/restores/model/Restore';
import type { RestoreTarget } from '@/entities/restores/model/RestoreTarget';
import { getApiClient, requireData } from '@/shared/api';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';

// The largest page the contract serves; a backup rarely has more restores than one page holds.
const RESTORE_PAGE_LIMIT = 100;

export const restoreApi = {
  // The restore dialog shows every restore of the backup, newest first, so this follows the
  // cursor to the last page.
  async listAllRestores(backupId: string): Promise<Restore[]> {
    return collectCursorPages(async (after: string | undefined) =>
      requireData(
        await getApiClient().GET('/restores', {
          params: { query: { backupId, limit: RESTORE_PAGE_LIMIT, after } },
        }),
      ),
    );
  },

  // Answers once the target is reachable, with the restore in progress.
  async startRestore(backupId: string, target: RestoreTarget): Promise<Restore> {
    return requireData(await getApiClient().POST('/restores', { body: { backupId, target } }));
  },

  async cancelRestore(restoreId: string): Promise<Restore> {
    return requireData(
      await getApiClient().POST('/restores/{restoreId}/cancel', {
        params: { path: { restoreId } },
      }),
    );
  },
};
