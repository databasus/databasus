import { getApplicationServer } from '@/shared/config';

// A proxy that terminates TLS still serves the page over https, so the address the browser talks to
// is what decides whether a key or an archive would cross the network unencrypted.
export const isInsecureTransfer = (): boolean =>
  new URL(getApplicationServer()).protocol === 'http:';
