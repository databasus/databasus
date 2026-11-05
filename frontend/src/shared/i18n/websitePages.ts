import type { Locale } from './dictionaries';

const WEBSITE_URL = 'https://databasus.com';

interface WebsitePage {
  // Path without leading or trailing slash; "" is the home page
  path: string;
  // Whether the website publishes the page in every language (TRANSLATED_PATHS in
  // website/app/i18n.ts). A test keeps this flag in sync with that list.
  isTranslated: boolean;
  // Translated pages keep the English heading ids, so an anchor works in every language
  anchor?: string;
}

// Every website page the application links to.
export const WEBSITE_PAGES = {
  home: { path: '', isTranslated: true },
  installation: { path: 'installation', isTranslated: true },
  security: { path: 'security', isTranslated: true },
  accessManagementGlobalSettings: {
    path: 'access-management',
    anchor: 'global-settings',
    isTranslated: true,
  },
  advancedConfigEmailSmtp: {
    path: 'advanced-config',
    anchor: 'email-smtp',
    isTranslated: true,
  },
  restoreVerification: { path: 'restore-verification', isTranslated: true },
  faqLocalhost: { path: 'faq/localhost', isTranslated: true },
  faqSupabase: { path: 'faq/supabase', isTranslated: true },
  storagesGoogleDrive: { path: 'storages/google-drive', isTranslated: true },
  storagesCloudflareR2: { path: 'storages/cloudflare-r2', isTranslated: true },
  notifiersSlack: { path: 'notifiers/slack', isTranslated: true },
  notifiersTeams: { path: 'notifiers/teams', isTranslated: true },
  notifiersMattermost: { path: 'notifiers/mattermost', isTranslated: false },
} satisfies Record<string, WebsitePage>;

type WebsitePageId = keyof typeof WEBSITE_PAGES;

// Opens the page in the selected language when the website publishes it in that language, and the
// English page otherwise. URLs end with a slash, the website's canonical form.
export const getWebsitePageUrl = (pageId: WebsitePageId, locale: Locale): string => {
  const page: WebsitePage = WEBSITE_PAGES[pageId];
  const languagePrefix = page.isTranslated && locale !== 'en' ? `/${locale}` : '';
  const path = page.path ? `/${page.path}/` : '/';
  const anchor = page.anchor ? `#${page.anchor}` : '';

  return `${WEBSITE_URL}${languagePrefix}${path}${anchor}`;
};
