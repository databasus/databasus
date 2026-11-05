import { type JSX } from 'react';

import { CLOUDFLARE_TURNSTILE_SITE_KEY } from '@/shared/config';

interface CloudflareTurnstileWidgetProps {
  containerRef: React.RefObject<HTMLDivElement | null>;
}

export function CloudflareTurnstileWidget({
  containerRef,
}: CloudflareTurnstileWidgetProps): JSX.Element | null {
  if (!CLOUDFLARE_TURNSTILE_SITE_KEY) {
    return null;
  }

  return <div ref={containerRef} className="mb-3" />;
}
