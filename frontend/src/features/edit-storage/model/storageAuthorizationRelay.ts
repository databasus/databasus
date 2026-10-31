export type StorageAuthorizationResponse =
  | { type: 'code'; state: string; code: string }
  | { type: 'refusal'; state: string };

export const isStorageAuthorizationState = (state: unknown): state is string =>
  typeof state === 'string' && /^[a-f0-9]{64}$/.test(state);

export const getStorageAuthorizationChannelName = (state: string) =>
  // eslint-disable-next-line i18next/no-literal-string -- attempt-specific browser channel name
  `databasus-storage-authorization:${state}`;

export const isStorageAuthorizationResponse = (
  message: unknown,
): message is StorageAuthorizationResponse => {
  if (!message || typeof message !== 'object') return false;

  const response = message as Record<string, unknown>;
  const keys = Object.keys(response).sort().join(',');

  return (
    isStorageAuthorizationState(response.state) &&
    ((response.type === 'refusal' && keys === 'state,type') ||
      (response.type === 'code' &&
        keys === 'code,state,type' &&
        typeof response.code === 'string' &&
        response.code.trim().length > 0))
  );
};
