import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { test } from '../browserTest';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';

test.use({ httpFixtures: [accountFixtures(), { scope: 'test' }] });

const CJK_FONT_FAMILY = /CJK|WenQuanYi|IPAGothic|IPAPGothic|Droid Sans Fallback/i;

test.describe('with a Traditional Chinese browser', () => {
  test.use({ locale: 'zh-TW' });

  test('opens the Chinese dictionary when no language is stored', async ({ page }) => {
    await page.goto('/');

    await expect(
      page.getByRole('button', { name: DICTIONARIES.zh.users.signIn.submit, exact: true }),
    ).toBeVisible();
  });

  test('renders Chinese text with a CJK font and requests no external font', async ({ page }) => {
    const externalFontUrls: string[] = [];
    page.on('request', (request) => {
      const isSameOrigin = new URL(request.url()).origin === new URL(page.url()).origin;
      if (request.resourceType() === 'font' && !isSameOrigin) externalFontUrls.push(request.url());
    });

    await page.goto('/');
    await page.evaluate(() => document.fonts.ready);

    const submit = page.getByRole('button', {
      name: DICTIONARIES.zh.users.signIn.submit,
      exact: true,
    });
    await expect(submit).toBeVisible();
    await submit.evaluate((button) => button.setAttribute('data-font-probe', ''));

    const devTools = await page.context().newCDPSession(page);
    await devTools.send('DOM.enable');
    await devTools.send('CSS.enable');
    const { root } = await devTools.send('DOM.getDocument');
    const { nodeId } = await devTools.send('DOM.querySelector', {
      nodeId: root.nodeId,
      selector: '[data-font-probe] span',
    });
    const { fonts } = await devTools.send('CSS.getPlatformFontsForNode', { nodeId });

    expect(fonts.map(({ familyName }) => familyName).join(', ')).toMatch(CJK_FONT_FAMILY);
    expect(externalFontUrls).toEqual([]);
  });
});

for (const { locale, regionalDate, regionalClock } of [
  { locale: 'en-GB', regionalDate: '01/10/2026', regionalClock: '00:00' },
  { locale: 'en-US', regionalDate: '10/01/2026', regionalClock: '12:00 AM' },
]) {
  test.describe(`with an ${locale} browser`, () => {
    test.use({ locale });

    test('orders the date and clock by the region under the English dictionary', async ({
      page,
    }) => {
      await openAccount({ page, locale: 'en' });
      await openAccountTab({ page, locale: 'en', tab: 'users' });

      await expect(page.getByRole('main')).toContainText(`${regionalDate} ${regionalClock}`);
    });
  });
}
