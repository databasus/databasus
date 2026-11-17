import type { IdentityProof } from '@/entities/instance-backups/model/IdentityProof';
import type { InstanceRecovery } from '@/entities/instance-backups/model/InstanceRecovery';
import type { ApiComponents, ReadOptions } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';

interface RecoveryUpload {
  archive: File;
  masterKey: string;
}

const toPath = (instanceRecoveryId: string) => ({ params: { path: { instanceRecoveryId } } });

export const instanceRecoveryApi = {
  // A string part of multipart/form-data has its line breaks rewritten to CRLF by the browser, so
  // the key goes as a Blob, whose bytes reach the server exactly, whitespace and line endings
  // included.
  async createRecovery(
    upload: RecoveryUpload,
    options: ReadOptions = {},
  ): Promise<InstanceRecovery> {
    return requireData(
      await getApiClient(options).POST('/instance-recoveries', {
        body: upload as unknown as ApiComponents['schemas']['InstanceRecoveryUpload'],
        bodySerializer: () => {
          const form = new FormData();
          form.append('masterKey', new Blob([upload.masterKey], { type: 'text/plain' }));
          form.append('archive', upload.archive, upload.archive.name);

          return form;
        },
      }),
    );
  },

  async getRecovery(
    instanceRecoveryId: string,
    options: ReadOptions = {},
  ): Promise<InstanceRecovery> {
    return requireData(
      await getApiClient(options).GET(
        '/instance-recoveries/{instanceRecoveryId}',
        toPath(instanceRecoveryId),
      ),
    );
  },

  async requestConfirmation(
    instanceRecoveryId: string,
  ): Promise<ApiComponents['schemas']['IdentityConfirmation']> {
    return requireData(
      await getApiClient().POST(
        '/instance-recoveries/{instanceRecoveryId}/confirmations',
        toPath(instanceRecoveryId),
      ),
    );
  },

  async applyRecovery(instanceRecoveryId: string, proof: IdentityProof): Promise<InstanceRecovery> {
    return requireData(
      await getApiClient().POST('/instance-recoveries/{instanceRecoveryId}/apply', {
        ...toPath(instanceRecoveryId),
        body: { isReplacementConfirmed: true, proof },
      }),
    );
  },

  async cancelRecovery(instanceRecoveryId: string): Promise<InstanceRecovery> {
    return requireData(
      await getApiClient().POST(
        '/instance-recoveries/{instanceRecoveryId}/cancel',
        toPath(instanceRecoveryId),
      ),
    );
  },

  async activateRecovery(instanceRecoveryId: string): Promise<InstanceRecovery> {
    return requireData(
      await getApiClient().POST('/instance-recoveries/{instanceRecoveryId}/activate', {
        ...toPath(instanceRecoveryId),
        body: { isOldInstanceStopped: true },
      }),
    );
  },
};
