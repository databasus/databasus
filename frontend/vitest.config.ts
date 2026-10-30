import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    fs: {
      // websitePages.test.ts reads ../website/app/i18n.ts to check the application's links to
      // website pages against the pages the website translates.
      allow: ['..'],
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.{ts,tsx}'],
  },
});
