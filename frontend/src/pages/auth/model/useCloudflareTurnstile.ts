import { useEffect, useRef, useState } from 'react';

import { CLOUDFLARE_TURNSTILE_SITE_KEY } from '@/shared/config';

declare global {
  interface Window {
    turnstile?: {
      render: (
        container: string | HTMLElement,
        options: {
          sitekey: string;
          callback: (token: string) => void;
          'error-callback'?: () => void;
          'expired-callback'?: () => void;
          theme?: 'light' | 'dark' | 'auto';
          size?: 'normal' | 'compact' | 'flexible';
          appearance?: 'always' | 'execute' | 'interaction-only';
        },
      ) => string;
      reset: (widgetId: string) => void;
      remove: (widgetId: string) => void;
      getResponse: (widgetId: string) => string | undefined;
    };
  }
}

interface UseCloudflareTurnstileReturn {
  containerRef: React.RefObject<HTMLDivElement | null>;
  token: string | undefined;
  resetCloudflareTurnstile: () => void;
}

const loadCloudflareTurnstileScript = (): Promise<void> => {
  if (!CLOUDFLARE_TURNSTILE_SITE_KEY) {
    return Promise.resolve();
  }

  return new Promise((resolve, reject) => {
    if (document.querySelector('script[src*="turnstile"]')) {
      resolve();
      return;
    }

    const script = document.createElement('script');
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
    script.async = true;
    script.defer = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Failed to load Cloudflare Turnstile'));
    document.head.appendChild(script);
  });
};

export function useCloudflareTurnstile(): UseCloudflareTurnstileReturn {
  const [token, setToken] = useState<string | undefined>(undefined);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const widgetIdRef = useRef<string | null>(null);

  useEffect(() => {
    if (!CLOUDFLARE_TURNSTILE_SITE_KEY || !containerRef.current) {
      return;
    }

    loadCloudflareTurnstileScript()
      .then(() => {
        if (!window.turnstile || !containerRef.current) {
          return;
        }

        try {
          const widgetId = window.turnstile.render(containerRef.current, {
            sitekey: CLOUDFLARE_TURNSTILE_SITE_KEY,
            callback: (receivedToken: string) => {
              setToken(receivedToken);
            },
            'error-callback': () => {
              setToken(undefined);
            },
            'expired-callback': () => {
              setToken(undefined);
            },
            theme: 'auto',
            size: 'normal',
            appearance: 'execute',
          });

          widgetIdRef.current = widgetId;
        } catch (error) {
          console.error('Failed to render Cloudflare Turnstile widget:', error);
        }
      })
      .catch((error) => {
        console.error('Failed to load Cloudflare Turnstile:', error);
      });

    return () => {
      if (widgetIdRef.current && window.turnstile) {
        window.turnstile.remove(widgetIdRef.current);
        widgetIdRef.current = null;
      }
    };
  }, []);

  const resetCloudflareTurnstile = () => {
    if (widgetIdRef.current && window.turnstile) {
      window.turnstile.reset(widgetIdRef.current);
      setToken(undefined);
    }
  };

  return {
    containerRef,
    token,
    resetCloudflareTurnstile,
  };
}
