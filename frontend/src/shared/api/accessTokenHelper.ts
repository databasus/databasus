const AUTHORIZED_USER_TOKEN_KEY = 'databasus_user_token';
const AUTHORIZED_USER_ID_KEY = 'databasus_user_id';

export const accessTokenHelper = {
  saveAccessToken: (token: string) => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    localStorage.setItem(AUTHORIZED_USER_TOKEN_KEY, token);
  },

  getAccessToken: (): string | undefined => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    return localStorage.getItem(AUTHORIZED_USER_TOKEN_KEY) || undefined;
  },

  clearStoredSession: () => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    localStorage.removeItem(AUTHORIZED_USER_TOKEN_KEY);
    localStorage.removeItem(AUTHORIZED_USER_ID_KEY);
  },

  saveUserId: (id: string) => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    localStorage.setItem(AUTHORIZED_USER_ID_KEY, id);
  },

  getUserId: (): string | undefined => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    return localStorage.getItem(AUTHORIZED_USER_ID_KEY) || undefined;
  },
};
