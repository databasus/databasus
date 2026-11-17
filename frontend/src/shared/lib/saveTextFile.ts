import { followFileLink } from '@/shared/lib/followFileLink';

// The browser reads the object URL after the click returns, so it is released a moment later.
const OBJECT_URL_LIFETIME_MS = 10_000;

export const saveTextFile = (content: string, fileName: string) => {
  const url = URL.createObjectURL(new Blob([content], { type: 'text/plain' }));

  followFileLink(url, fileName);
  setTimeout(() => URL.revokeObjectURL(url), OBJECT_URL_LIFETIME_MS);
};
