import type { IdentityProof } from '@/entities/instance-backups/model/IdentityProof';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { getApiClient, requireData } from '@/shared/api';
import { getApplicationServer } from '@/shared/config';

type Schemas = ApiComponents['schemas'];

export const masterKeyApi = {
  async requestConfirmation(): Promise<Schemas['IdentityConfirmation']> {
    return requireData(await getApiClient().POST('/master-key-confirmations'));
  },

  async createExport(
    proof: IdentityProof,
    isInsecureTransferAcknowledged: boolean,
  ): Promise<Schemas['MasterKeyExport']> {
    return requireData(
      await getApiClient().POST('/master-key-exports', {
        body: { proof, isInsecureTransferAcknowledged },
      }),
    );
  },

  // A browser link cannot carry the session, so the single-use token is the credential.
  getDownloadUrl(token: string): string {
    const query = new URLSearchParams({ token });

    return `${getApplicationServer()}${API_BASE_PATH}/master-key-downloads?${query.toString()}`;
  },

  async acknowledge(isSaved: boolean): Promise<Schemas['MasterKeyAcknowledgment']> {
    return requireData(
      await getApiClient().PUT('/master-key-acknowledgments', { body: { isSaved } }),
    );
  },
};
