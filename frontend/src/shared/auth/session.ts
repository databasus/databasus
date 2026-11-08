export interface SessionSnapshot {
  readonly token: string | undefined;
  readonly userId: string | undefined;
  readonly generation: number;
}

export interface AuthIntent {
  readonly generation: number;
  readonly revision: number;
}

const TOKEN_KEY = 'databasus_user_token';
const USER_KEY = 'databasus_user_id';
const SCOPED_SELECTION_KEY = /^selected[A-Z][A-Za-z]*Id_/;
const listeners = new Set<() => void>();
let snapshot: SessionSnapshot = { token: undefined, userId: undefined, generation: 0 };
let intentRevision = 0;

export const getSessionSnapshot = (): SessionSnapshot => {
  const token = globalThis.localStorage?.getItem(TOKEN_KEY) || undefined;
  const userId = globalThis.localStorage?.getItem(USER_KEY) || undefined;
  if (token !== snapshot.token || userId !== snapshot.userId) {
    snapshot = { token, userId, generation: snapshot.generation + 1 };
    intentRevision++;
  }
  return snapshot;
};

export const subscribeSession = (listener: () => void): (() => void) => {
  listeners.add(listener);
  let isSubscribed = true;
  return () => {
    if (!isSubscribed) return;
    isSubscribed = false;
    listeners.delete(listener);
  };
};

export const isCurrentSession = (session: SessionSnapshot): boolean =>
  session === getSessionSnapshot();

const removeScopedSelections = () => {
  const storage = globalThis.localStorage;
  if (!storage) return;

  const storedKeys = Array.from({ length: storage.length }, (_, index) => storage.key(index));
  for (const storedKey of storedKeys)
    if (storedKey && SCOPED_SELECTION_KEY.test(storedKey)) storage.removeItem(storedKey);
};

const publishSession = (identity?: { token: string; userId: string }) => {
  const generation = getSessionSnapshot().generation + 1;
  if (identity) {
    globalThis.localStorage?.setItem(TOKEN_KEY, identity.token);
    globalThis.localStorage?.setItem(USER_KEY, identity.userId);
  } else {
    globalThis.localStorage?.removeItem(TOKEN_KEY);
    globalThis.localStorage?.removeItem(USER_KEY);
  }

  globalThis.localStorage?.removeItem('selected_workspace_id');
  removeScopedSelections();
  globalThis.localStorage?.removeItem('logicalBackupListFilter');
  globalThis.localStorage?.removeItem('physicalBackupsFilters');
  globalThis.sessionStorage?.removeItem('databasus_oauth_sign_in_attempt');

  snapshot = { token: identity?.token, userId: identity?.userId, generation };
  intentRevision++;
  for (const listener of listeners) listener();
};

export const activateSession = (identity: { token: string; userId: string }) =>
  publishSession(identity);

export const invalidateSession = (session = getSessionSnapshot()): boolean => {
  if (!session.token || !isCurrentSession(session)) return false;
  publishSession();
  return true;
};

export const getAuthIntent = (): AuthIntent => ({
  generation: getSessionSnapshot().generation,
  revision: intentRevision,
});
export const beginAuthIntent = (): AuthIntent => {
  globalThis.sessionStorage?.removeItem('databasus_oauth_sign_in_attempt');
  intentRevision++;

  return getAuthIntent();
};
export const isCurrentAuthIntent = (intent: AuthIntent): boolean =>
  intent.generation === getSessionSnapshot().generation && intent.revision === intentRevision;
export const abandonAuthIntent = (intent: AuthIntent): void => {
  if (isCurrentAuthIntent(intent)) {
    globalThis.sessionStorage?.removeItem('databasus_oauth_sign_in_attempt');
    intentRevision++;
  }
};
export const acceptAuthSession = ({
  intent,
  session,
}: {
  intent: AuthIntent;
  session: { token: string; userId: string };
}): boolean => {
  if (!isCurrentAuthIntent(intent)) return false;
  activateSession(session);
  return true;
};
