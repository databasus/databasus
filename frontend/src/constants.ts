interface RuntimeConfig {
  GITHUB_CLIENT_ID?: string;
  GOOGLE_CLIENT_ID?: string;
  IS_EMAIL_CONFIGURED?: string;
  CLOUDFLARE_TURNSTILE_SITE_KEY?: string;
  CONTAINER_ARCH?: string;
}

declare global {
  interface Window {
    __RUNTIME_CONFIG__?: RuntimeConfig;
  }
}

// Tests import this module outside a browser, where window does not exist.
const runtimeConfig = globalThis.window?.__RUNTIME_CONFIG__;

export function getApplicationServer() {
  const origin = window.location.origin;
  const url = new URL(origin);

  const isDevelopment = import.meta.env.MODE === 'development';

  if (isDevelopment) {
    return `${url.protocol}//${url.hostname}:4005`;
  } else {
    return `${url.protocol}//${url.hostname}:${url.port || (url.protocol === 'https:' ? '443' : '80')}`;
  }
}

// eslint-disable-next-line i18next/no-literal-string -- version identifier of a development build
export const APP_VERSION = (import.meta.env.VITE_APP_VERSION as string) || 'dev';

export const GITHUB_CLIENT_ID =
  runtimeConfig?.GITHUB_CLIENT_ID || import.meta.env.VITE_GITHUB_CLIENT_ID || '';

export const GOOGLE_CLIENT_ID =
  runtimeConfig?.GOOGLE_CLIENT_ID || import.meta.env.VITE_GOOGLE_CLIENT_ID || '';

export const IS_EMAIL_CONFIGURED =
  runtimeConfig?.IS_EMAIL_CONFIGURED === 'true' ||
  import.meta.env.VITE_IS_EMAIL_CONFIGURED === 'true';

export const CLOUDFLARE_TURNSTILE_SITE_KEY =
  runtimeConfig?.CLOUDFLARE_TURNSTILE_SITE_KEY ||
  import.meta.env.VITE_CLOUDFLARE_TURNSTILE_SITE_KEY ||
  '';

// eslint-disable-next-line i18next/no-literal-string -- CPU architecture identifiers
const archMap: Record<string, string> = { amd64: 'x64', arm64: 'arm64' };
const rawArch = runtimeConfig?.CONTAINER_ARCH ?? '';
export const CONTAINER_ARCH = archMap[rawArch] || rawArch;

export function getOAuthRedirectUri(): string {
  return `${window.location.origin}/auth/callback`;
}
