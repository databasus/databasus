import { API_BASE_PATH, API_ERROR_CODES, ApiError, type ApiComponents } from '@/shared/api';
import { getApiClient, requireData, requireSuccess } from '@/shared/api';
import { getApplicationServer } from '@/shared/config';

type Schemas = ApiComponents['schemas'];

export interface MasterKeyFile {
  content: string;
  fileName: string;
}

const getDownloadUrl = (token: string): string => {
  const query = new URLSearchParams({ token });

  return `${getApplicationServer()}${API_BASE_PATH}/master-key-downloads?${query.toString()}`;
};

// The token is the credential, so the request carries no session: a refused token must not read
// as an expired session and sign root admin out.
const readKeyFile = async (token: string): Promise<string> => {
  let response: Response;
  try {
    response = await fetch(getDownloadUrl(token), { cache: 'no-store' });
  } catch {
    throw new ApiError({ code: API_ERROR_CODES.networkUnreachable });
  }

  if (!response.ok) {
    const problem: unknown = await response.json().catch(() => undefined);
    requireSuccess({ response, error: problem });
  }

  return response.text();
};

export const masterKeyApi = {
  async requestConfirmation(): Promise<Schemas['IdentityConfirmation']> {
    return requireData(await getApiClient().POST('/master-key-confirmations'));
  },

  // One export serves both the copy and the download, so the page keeps the key it read once.
  async getKeyFile(code?: string): Promise<MasterKeyFile> {
    const keyExport = requireData(
      await getApiClient().POST('/master-key-exports', { body: { code } }),
    );

    return { content: await readKeyFile(keyExport.token), fileName: keyExport.fileName };
  },

  async acknowledge(): Promise<Schemas['MasterKeyAcknowledgment']> {
    return requireData(
      await getApiClient().PUT('/master-key-acknowledgments', { body: { isSaved: true } }),
    );
  },
};
