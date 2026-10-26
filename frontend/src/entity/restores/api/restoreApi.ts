import { getApplicationServer } from '../../../constants';
import { apiHelper } from '../../../shared/api/apiHelper';
import RequestOptions from '../../../shared/api/RequestOptions';
import type { Restore } from '../model/Restore';
import type { RestoreBackupRequest } from '../model/RestoreTarget';

export const restoreApi = {
  async getRestores(backupId: string) {
    return apiHelper.fetchGetJson<Restore[]>(
      `${getApplicationServer()}/api/v1/restores/${backupId}`,
      undefined,
      true,
    );
  },

  async restoreBackup(backupId: string, request: RestoreBackupRequest) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));

    return apiHelper.fetchPostJson<{ message: string }>(
      `${getApplicationServer()}/api/v1/restores/${backupId}/restore`,
      requestOptions,
    );
  },

  async cancelRestore(restoreId: string) {
    return apiHelper.fetchPostRaw(`${getApplicationServer()}/api/v1/restores/cancel/${restoreId}`);
  },
};
