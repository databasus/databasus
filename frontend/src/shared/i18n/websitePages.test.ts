import { describe, expect, it } from 'vitest';

import websiteI18nSource from '../../../../website/app/i18n.ts?raw';
import { WEBSITE_PAGES, getWebsitePageUrl } from './websitePages';

const readTranslatedPaths = (source: string): string[] => {
  const list = source.match(/TRANSLATED_PATHS\s*=\s*\[([\s\S]*?)\]\s*as const/);
  if (!list) {
    throw new Error('TRANSLATED_PATHS not found in website/app/i18n.ts');
  }

  return [...list[1].matchAll(/"([^"]*)"/g)].map((match) => match[1]);
};

describe('WEBSITE_PAGES', () => {
  const translatedPaths = readTranslatedPaths(websiteI18nSource);

  // A page the website stops translating would otherwise 404 in the application, and a page it
  // starts translating would keep opening in English.
  it.each(Object.entries(WEBSITE_PAGES))(
    'marks %s as translated exactly when the website translates it',
    (_, page) => {
      expect(page.isTranslated).toBe(translatedPaths.includes(page.path));
    },
  );
});

describe('getWebsitePageUrl', () => {
  it('opens a translated page in the selected language', () => {
    expect(getWebsitePageUrl('installation', 'ru')).toBe('https://databasus.com/ru/installation/');
  });

  it('opens a translated page in Indonesian', () => {
    expect(getWebsitePageUrl('installation', 'id')).toBe('https://databasus.com/id/installation/');
  });

  it('opens the English page for English', () => {
    expect(getWebsitePageUrl('installation', 'en')).toBe('https://databasus.com/installation/');
  });

  it('opens the English page when the website does not translate it', () => {
    expect(getWebsitePageUrl('notifiersMattermost', 'ru')).toBe(
      'https://databasus.com/notifiers/mattermost/',
    );
    expect(getWebsitePageUrl('notifiersMattermost', 'id')).toBe(
      'https://databasus.com/notifiers/mattermost/',
    );
  });

  it('keeps the anchor of a section in every language', () => {
    expect(getWebsitePageUrl('accessManagementGlobalSettings', 'ru')).toBe(
      'https://databasus.com/ru/access-management/#global-settings',
    );
    expect(getWebsitePageUrl('accessManagementGlobalSettings', 'id')).toBe(
      'https://databasus.com/id/access-management/#global-settings',
    );
    expect(getWebsitePageUrl('accessManagementGlobalSettings', 'en')).toBe(
      'https://databasus.com/access-management/#global-settings',
    );
  });

  it('links the SMTP section of the configuration page', () => {
    expect(getWebsitePageUrl('advancedConfigEmailSmtp', 'en')).toBe(
      'https://databasus.com/advanced-config/#email-smtp',
    );
    expect(getWebsitePageUrl('advancedConfigEmailSmtp', 'fr')).toBe(
      'https://databasus.com/fr/advanced-config/#email-smtp',
    );
  });

  it('links the home page with and without a language', () => {
    expect(getWebsitePageUrl('home', 'en')).toBe('https://databasus.com/');
    expect(getWebsitePageUrl('home', 'ru')).toBe('https://databasus.com/ru/');
  });
});
