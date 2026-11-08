import AxeBuilder from '@axe-core/playwright';
import { expect, type Page, type Locator } from '@playwright/test';

import { DICTIONARIES, type Locale } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, smtpNotifier } from '../applicationFixtures';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';

const assertEditorReflow = async (editor: Locator) => {
  const geometry = await editor.evaluate((region) => {
    const bounds = Array.from(
      region.querySelectorAll<HTMLElement>('button, label[for], input, [role="alert"]'),
    )
      .filter((element) => element.getClientRects().length > 0)
      .map((element) => ({ text: element.textContent, bounds: element.getBoundingClientRect() }));

    const overlaps: string[] = [];
    for (let first = 0; first < bounds.length; first++)
      for (let second = first + 1; second < bounds.length; second++) {
        const left = bounds[first],
          right = bounds[second];
        if (
          Math.min(left.bounds.right, right.bounds.right) -
            Math.max(left.bounds.left, right.bounds.left) >
            1 &&
          Math.min(left.bounds.bottom, right.bounds.bottom) -
            Math.max(left.bounds.top, right.bounds.top) >
            1
        )
          overlaps.push(`${left.text} / ${right.text}`);
      }

    return {
      overlaps,
      hasOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    };
  });
  expect(geometry).toEqual({ overlaps: [], hasOverflow: false });
};

const openEditor = async ({
  page,
  locale = 'en',
  theme = 'light',
}: {
  page: Page;
  locale?: Locale;
  theme?: string;
}) => {
  await page.clock.setFixedTime(new Date('2026-10-18T12:00:00Z'));
  await page.addInitScript(
    ({ locale, theme }) => {
      localStorage.setItem('databasus_user_token', 'synthetic-session');
      localStorage.setItem('databasus_user_id', 'owner-id');
      localStorage.setItem('databasus-locale', locale);
      localStorage.setItem('databasus-theme', theme);
    },
    { locale, theme },
  );

  await page.goto('/');
  const dictionary = DICTIONARIES[locale];
  const menu = page.getByRole('button', { name: dictionary.app.navigation.menu, exact: true });
  if (await menu.isVisible()) await menu.click();

  await page.getByRole('link', { name: dictionary.app.navigation.notifiers, exact: true }).click();
  const notifier = page.getByRole('button', { name: smtpNotifier.name, exact: true });
  await expect(notifier).toBeVisible();
  await notifier.click();
  await page
    .getByRole('button', { name: dictionary.common.actions.edit, exact: true })
    .last()
    .click();

  const editor = page.getByRole('region', {
    name: dictionary.notifiers.config.settingsTitle,
    exact: true,
  });
  await expect(editor.getByLabel(dictionary.notifiers.email.smtpHost, { exact: true })).toHaveValue(
    'smtp.example.com',
  );

  await page.evaluate(() => document.fonts.ready);
  return editor;
};

test('tests and saves explicitly while preserving an omitted stored SMTP password', async ({
  page,
  boundary,
}) => {
  const editor = await openEditor({ page });
  await editor.getByRole('textbox', { name: 'From', exact: true }).fill('new@example.com');
  await editor.getByRole('button', { name: 'Send test notification', exact: true }).click();
  await editor.getByRole('button', { name: 'Save', exact: true }).click();

  await expect
    .poll(() => boundary.requests.filter((request) => request.method === 'PUT').length)
    .toBe(1);

  const update = boundary.requests.find((request) => request.method === 'PUT')!;
  expect(update.body).toMatchObject({ settings: { from: 'new@example.com' } });
  expect((update.body as { settings: object }).settings).not.toHaveProperty('smtpPassword');
});

test('preserves values, expanded settings and successful testing across locale and theme switches', async ({
  page,
  boundary,
}) => {
  const editor = await openEditor({ page });
  await editor.getByRole('textbox', { name: 'SMTP host', exact: true }).fill('draft.example.com');
  const disclosure = editor.getByRole('button', { name: 'Advanced settings', exact: true });
  await disclosure.focus();
  await page.keyboard.press('Space');
  await expect(disclosure).toHaveAttribute('aria-expanded', 'true');

  await editor.getByRole('button', { name: 'Send test notification', exact: true }).click();
  await expect(editor.getByRole('button', { name: 'Save', exact: true })).toBeVisible();

  const language = page.getByRole('button', { name: /Language:/ });
  await language.focus();
  await page.keyboard.press('ArrowDown');
  await expect
    .poll(() => page.getByRole('menu').evaluate((menu) => menu.contains(document.activeElement)))
    .toBe(true);
  await page.keyboard.press('Home');
  await expect(
    page.getByRole('menuitemradio', { name: DICTIONARIES.en.language.nativeName, exact: true }),
  ).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await expect(
    page.getByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName, exact: true }),
  ).toBeFocused();

  await page.keyboard.press('Enter');
  const russianLanguage = page.getByRole('button', {
    name: DICTIONARIES.ru.app.languageControl.title.replace(
      '{{language}}',
      DICTIONARIES.ru.language.nativeName,
    ),
    exact: true,
  });
  await expect(russianLanguage).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await expect(
    page.getByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName, exact: true }),
  ).toHaveAttribute('aria-checked', 'true');
  await assertEnlargedTextReflow(page);

  await page.keyboard.press('Escape');
  await expect(russianLanguage).toBeFocused();
  await expect(russianLanguage).toHaveAttribute('aria-expanded', 'false');

  const russianEditor = page.getByRole('region', {
    name: DICTIONARIES.ru.notifiers.config.settingsTitle,
    exact: true,
  });
  await expect(
    russianEditor.getByLabel(DICTIONARIES.ru.notifiers.email.smtpHost, { exact: true }),
  ).toHaveValue('draft.example.com');
  await expect(
    russianEditor.getByRole('button', { name: DICTIONARIES.ru.common.actions.save, exact: true }),
  ).toBeVisible();

  const theme = page.getByRole('button', {
    name: new RegExp(DICTIONARIES.ru.app.theme.title.split(':')[0]),
  });
  await theme.focus();
  await page.keyboard.press('Enter');
  await page
    .getByRole('menuitemradio', { name: new RegExp(DICTIONARIES.ru.app.theme.dark) })
    .click();
  await expect(page.locator('html')).toHaveClass('dark');

  expect(boundary.requests.filter((request) => request.path === '/notifier-tests')).toHaveLength(1);
  await expect(
    russianEditor.getByRole('button', {
      name: DICTIONARIES.ru.notifiers.advancedSettings,
      exact: true,
    }),
  ).toHaveAttribute('aria-expanded', 'true');
});

test.describe('SMTP delivery failure', () => {
  const diagnostic = '<img src=x onerror=alert(1)> {{target}}';
  test.use({
    httpFixtures: [
      [
        ...applicationFixtures().filter((fixture) => fixture.path !== '/notifier-tests'),
        apiResponse({
          method: 'post',
          path: '/notifier-tests',
          status: 200,
          body: { isDelivered: false, error: diagnostic },
        }),
      ],
      { scope: 'test' },
    ],
  });
  test('keeps the SMTP draft and renders failed-delivery diagnostics as literal text', async ({
    page,
    boundary,
  }) => {
    const editor = await openEditor({ page });
    await editor.getByRole('textbox', { name: 'SMTP host', exact: true }).fill('draft.example.com');
    await editor.getByRole('button', { name: 'Send test notification', exact: true }).click();

    await expect(editor.getByRole('alert')).toContainText(diagnostic);
    expect(await editor.getByRole('alert').locator('img').count()).toBe(0);
    await expect(editor.getByRole('textbox', { name: 'SMTP host', exact: true })).toHaveValue(
      'draft.example.com',
    );
    expect(boundary.requests.filter((request) => request.path === '/notifier-tests')).toHaveLength(
      1,
    );
  });
});

test('exposes SMTP fields, associated validation and shared keyboard focus without accessibility violations', async ({
  page,
}) => {
  const editor = await openEditor({ page });
  const target = editor.getByRole('textbox', { name: 'Target email', exact: true });
  await target.fill('invalid');
  await target.press('Tab');
  await expect(target).toHaveAttribute('aria-invalid', 'true');
  await expect(target).toHaveAccessibleDescription(/valid email/);

  const disclosure = editor.getByRole('button', { name: 'Advanced settings', exact: true });
  await disclosure.focus();
  await page.keyboard.press('Enter');
  await expect(disclosure).toBeFocused();
  await expect(disclosure).toHaveCSS('outline-style', 'solid');

  await page.evaluate(() =>
    Promise.allSettled(document.getAnimations().map((animation) => animation.finished)),
  );

  const results = await new AxeBuilder({ page })
    .disableRules(['color-contrast'])
    .include('section[aria-label="Notifier settings"]')
    .analyze();
  expect(results.violations).toEqual([]);

  await page.getByRole('button', { name: /Language:/ }).focus();
  await page.keyboard.press('ArrowDown');
  await page
    .getByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName, exact: true })
    .click();

  const translatedTarget = page.getByLabel(DICTIONARIES.ru.notifiers.email.targetEmail, {
    exact: true,
  });
  await expect(translatedTarget).toHaveValue('invalid');
  await expect(translatedTarget).toHaveAccessibleDescription(
    new RegExp(DICTIONARIES.ru.notifiers.email.validation.targetEmail),
  );
});

test('contains confirmation focus and restores it to the invoking remove control', async ({
  page,
}) => {
  const editor = await openEditor({ page });
  await editor.getByRole('button', { name: 'Cancel', exact: true }).click();
  const remove = page.getByRole('button', { name: 'Remove', exact: true });
  await remove.click();

  const dialog = page.getByRole('dialog', { name: 'Confirmation' });
  await expect(dialog).toBeVisible();
  await dialog.getByRole('button', { name: 'Cancel', exact: true }).focus();

  for (let index = 0; index < 6; index++) {
    await page.keyboard.press('Tab');
    expect(await dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  }
  await assertEnlargedTextReflow(page);

  await page.keyboard.press('Escape');
  await expect(dialog).not.toBeVisible();
  await expect(remove).toBeFocused();
});

for (const reference of [
  { name: 'smtp-light-en-desktop', locale: 'en', theme: 'light', width: 1280, hasError: false },
  {
    name: 'smtp-dark-ru-desktop-validation',
    locale: 'ru',
    theme: 'dark',
    width: 1280,
    hasError: true,
  },
  { name: 'smtp-light-fr-narrow', locale: 'fr', theme: 'light', width: 320, hasError: false },
  { name: 'smtp-dark-zh-narrow', locale: 'zh', theme: 'dark', width: 320, hasError: false },
] as const) {
  test(`reference ${reference.name}`, async ({ page, boundary }, testInfo) => {
    await page.setViewportSize({ width: reference.width, height: 1200 });
    const editor = await openEditor({ page, locale: reference.locale, theme: reference.theme });
    const dictionary = DICTIONARIES[reference.locale];

    if (reference.hasError) {
      const target = editor.getByLabel(dictionary.notifiers.email.targetEmail, { exact: true });
      await target.fill('invalid');
      await target.press('Tab');
      await expect(target).toHaveAttribute('aria-invalid', 'true');
    }

    await expect(editor).toHaveScreenshot(`${reference.name}.png`);
    const layout = await page.evaluate(() => ({
      width: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
    }));
    expect(layout.scrollWidth).toBeLessThanOrEqual(layout.width);

    expect(
      boundary.requests.some((request) => /fonts\.(?:googleapis|gstatic)/.test(request.path)),
    ).toBe(false);
    expect(await page.evaluate(() => document.fonts.check('16px Jost'))).toBe(true);

    const fonts = boundary.resourceUrls.filter((url) => url.includes('.woff2'));
    expect(fonts.length).toBeGreaterThan(0);
    expect(fonts.every((url) => new URL(url).origin === new URL(page.url()).origin)).toBe(true);

    const cdp = await page.context().newCDPSession(page);
    await cdp.send('DOM.enable');
    await cdp.send('CSS.enable');
    const { root } = await cdp.send('DOM.getDocument');
    const { nodeId } = await cdp.send('DOM.querySelector', {
      nodeId: root.nodeId,
      selector: 'section label[for$="-smtpHost"]',
    });
    const rendered = await cdp.send('CSS.getPlatformFontsForNode', { nodeId });

    expect(
      rendered.fonts.some(
        (font) => font.isCustomFont && font.familyName.includes('Jost') && font.glyphCount > 0,
      ),
    ).toBe(true);

    if (reference.locale === 'zh')
      expect(rendered.fonts.some((font) => !font.isCustomFont && font.glyphCount > 0)).toBe(true);
    await testInfo.attach('rendered-fonts', {
      body: JSON.stringify(rendered.fonts),
      contentType: 'application/json',
    });
    await cdp.detach();

    if (reference.width === 320) {
      await editor
        .getByRole('button', { name: dictionary.notifiers.advancedSettings, exact: true })
        .click();

      const target = editor.getByLabel(dictionary.notifiers.email.targetEmail, { exact: true });
      const originalSize = parseFloat(
        await target.evaluate((element) => getComputedStyle(element).fontSize),
      );

      await page.evaluate(() => {
        const originalSizes = Array.from(
          document.querySelectorAll<HTMLElement>('section[aria-label] *'),
        ).map((element) => ({ element, size: parseFloat(getComputedStyle(element).fontSize) }));
        for (const { element, size } of originalSizes) element.style.fontSize = `${size * 2}px`;
      });

      await expect(target).toHaveCSS('font-size', `${originalSize * 2}px`);
      const overlappingLabels = await editor.locator('label[for]').evaluateAll((labels) =>
        labels
          .filter((label) => {
            const control = document.getElementById(label.getAttribute('for')!);
            if (!control || control.matches('[type=checkbox]')) return false;
            const labelBounds = label.getBoundingClientRect();
            const controlBounds = control.getBoundingClientRect();
            return (
              labelBounds.bottom > controlBounds.top + 1 &&
              labelBounds.right > controlBounds.left + 1
            );
          })
          .map((label) => label.textContent),
      );
      expect(overlappingLabels).toEqual([]);

      const cancel = editor.getByRole('button', {
        name: dictionary.common.actions.cancel,
        exact: true,
      });
      await cancel.scrollIntoViewIfNeeded();
      await expect(cancel).toBeVisible();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
        ),
      ).toBe(true);
    }
  });
}

test('expires an existing success notice at its original deadline and uses the selected language for later notices', async ({
  page,
}) => {
  await page.clock.install({ time: new Date('2026-10-18T12:00:00Z') });
  const editor = await openEditor({ page });
  await editor.getByLabel('SMTP host', { exact: true }).fill('first.example.com');
  await editor.getByRole('button', { name: 'Send test notification', exact: true }).click();
  await expect(
    page.getByText(DICTIONARIES.en.notifiers.testNotification.sent.title, { exact: true }),
  ).toBeVisible();

  await page.clock.runFor(2_000);
  await page.getByRole('button', { name: /Language:/ }).focus();
  await page.keyboard.press('ArrowDown');
  await page
    .getByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName, exact: true })
    .click();

  await expect(
    page.getByText(DICTIONARIES.en.notifiers.testNotification.sent.title, { exact: true }),
  ).toBeVisible();
  await page.clock.runFor(2_100);
  await expect(
    page.getByText(DICTIONARIES.en.notifiers.testNotification.sent.title, { exact: true }),
  ).not.toBeVisible();

  const russianEditor = page.getByRole('region', {
    name: DICTIONARIES.ru.notifiers.config.settingsTitle,
    exact: true,
  });
  await russianEditor
    .getByLabel(DICTIONARIES.ru.notifiers.email.smtpHost, { exact: true })
    .fill('second.example.com');
  await russianEditor
    .getByRole('button', { name: DICTIONARIES.ru.notifiers.testNotification.send, exact: true })
    .click();
  await expect(
    page.getByText(DICTIONARIES.ru.notifiers.testNotification.sent.title, { exact: true }),
  ).toBeVisible();

  await russianEditor
    .getByRole('button', { name: DICTIONARIES.ru.common.actions.cancel, exact: true })
    .click();
  await expect(
    page.getByText(DICTIONARIES.ru.notifiers.testNotification.sent.title, { exact: true }),
  ).not.toBeVisible();
});

for (const locale of ['fr', 'zh'] as const) {
  test(`keeps validation, testing and saving reachable with enlarged ${locale} text at 320 pixels`, async ({
    page,
    boundary,
  }) => {
    await page.setViewportSize({ width: 320, height: 1200 });
    const editor = await openEditor({ page, locale });
    const dictionary = DICTIONARIES[locale];
    const target = editor.getByLabel(dictionary.notifiers.email.targetEmail, { exact: true });

    await editor
      .getByLabel(dictionary.notifiers.email.smtpHost, { exact: true })
      .fill('changed.example.com');
    await target.fill('invalid');
    await target.press('Tab');
    const error = editor.getByText(dictionary.notifiers.email.validation.targetEmail, {
      exact: true,
    });
    await expect(error).toBeVisible();

    const originalSize = parseFloat(
      await target.evaluate((element) => getComputedStyle(element).fontSize),
    );
    await page.evaluate(() => {
      const sizes = Array.from(document.querySelectorAll<HTMLElement>('section[aria-label] *')).map(
        (element) => ({ element, size: parseFloat(getComputedStyle(element).fontSize) }),
      );
      for (const { element, size } of sizes) element.style.fontSize = `${size * 2}px`;
    });
    await expect(target).toHaveCSS('font-size', `${originalSize * 2}px`);

    await error.scrollIntoViewIfNeeded();
    await expect(error).toBeInViewport();
    const testControl = editor.getByRole('button', {
      name: dictionary.notifiers.testNotification.send,
      exact: true,
    });
    await expect(testControl).toBeDisabled();
    await testControl.scrollIntoViewIfNeeded();
    await expect(testControl).toBeInViewport();
    await assertEditorReflow(editor);

    await target.fill('ops@example.com');
    await target.press('Tab');
    await expect(error).not.toBeVisible();
    await testControl.scrollIntoViewIfNeeded();
    await expect(testControl).toBeInViewport();
    await testControl.click();

    const successNotice = page.getByRole('status').filter({
      hasText: dictionary.notifiers.testNotification.sent.title,
    });
    await expect(successNotice).toBeVisible();
    await successNotice
      .getByRole('button', { name: dictionary.common.actions.close, exact: true })
      .click();
    await expect(successNotice).not.toBeVisible();

    const save = editor.getByRole('button', { name: dictionary.common.actions.save, exact: true });
    await expect(save).toBeVisible();
    await save.scrollIntoViewIfNeeded();
    await expect(save).toBeInViewport();
    await assertEditorReflow(editor);
    await save.click();

    await expect
      .poll(() => boundary.requests.filter((request) => request.method === 'PUT').length)
      .toBe(1);
    expect(boundary.requests.filter((request) => request.path === '/notifier-tests')).toHaveLength(
      1,
    );
  });
}

test('preserves the open sidebar and notifier draft when language is selected at 320 pixels', async ({
  page,
  boundary,
}) => {
  await page.setViewportSize({ width: 320, height: 1200 });
  const editor = await openEditor({ page });
  await editor
    .getByLabel(DICTIONARIES.en.notifiers.email.smtpHost, { exact: true })
    .fill('draft.example.com');
  await editor
    .getByRole('button', { name: DICTIONARIES.en.notifiers.advancedSettings, exact: true })
    .click();

  await page
    .getByRole('button', { name: DICTIONARIES.en.app.navigation.menu, exact: true })
    .click();
  await page.mouse.move(319, 1100);

  const sidebar = page.getByRole('dialog');
  const language = sidebar.getByRole('button', { name: /Language:/ });
  const documentation = sidebar.getByRole('link', {
    name: DICTIONARIES.en.app.navigation.documentation,
    exact: true,
  });
  expect((await documentation.boundingBox())!.y).toBeGreaterThan(page.viewportSize()!.height - 200);
  for (const { name, firstItem } of [
    { name: /Language:/, firstItem: DICTIONARIES.en.language.nativeName },
    { name: /Theme:/, firstItem: DICTIONARIES.en.app.theme.light },
  ]) {
    const control = sidebar.getByRole('button', { name });
    await control.focus();
    await page.keyboard.press('ArrowDown');

    const menu = page.getByRole('menu').filter({
      has: page.getByRole('menuitemradio', { name: firstItem, exact: true }),
    });
    await expect(menu).toBeVisible();
    await expect
      .poll(() => menu.evaluate((element) => element.contains(document.activeElement)))
      .toBe(true);

    await page.keyboard.press('Escape');
    await expect(control).toHaveAttribute('aria-expanded', 'false');
    await expect(sidebar).toBeVisible();
    await expect(control).toBeFocused();
    await control.press('ArrowDown');
    await expect(control).toHaveAttribute('aria-expanded', 'true');
    await control.press('Escape');
    await expect(control).toHaveAttribute('aria-expanded', 'false');
    await expect(sidebar).toBeVisible();
    await expect(control).toBeFocused();
  }
  await language.focus();
  await page.keyboard.press('ArrowDown');
  const menu = page.getByRole('menu').filter({
    has: page.getByRole('menuitemradio', {
      name: DICTIONARIES.en.language.nativeName,
      exact: true,
    }),
  });
  await expect
    .poll(() => menu.evaluate((element) => element.contains(document.activeElement)))
    .toBe(true);

  await page.keyboard.press('Home');
  await expect(
    page.getByRole('menuitemradio', { name: DICTIONARIES.en.language.nativeName, exact: true }),
  ).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await expect(
    page.getByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName, exact: true }),
  ).toBeFocused();
  await page.keyboard.press('Enter');

  const dictionary = DICTIONARIES.ru;
  await expect(sidebar).toBeVisible();
  await expect(
    sidebar.getByRole('link', { name: dictionary.app.navigation.notifiers, exact: true }),
  ).toBeVisible();
  await expect(
    sidebar.getByRole('link', { name: dictionary.app.navigation.documentation, exact: true }),
  ).toBeVisible();
  await expect(
    sidebar.getByRole('button', {
      name: dictionary.app.languageControl.title.replace(
        '{{language}}',
        dictionary.language.nativeName,
      ),
      exact: true,
    }),
  ).toBeFocused();

  await sidebar.getByRole('button', { name: dictionary.common.actions.close, exact: true }).click();
  const translatedEditor = page.getByRole('region', {
    name: dictionary.notifiers.config.settingsTitle,
    exact: true,
  });
  await expect(
    translatedEditor.getByLabel(dictionary.notifiers.email.smtpHost, { exact: true }),
  ).toHaveValue('draft.example.com');
  await expect(
    translatedEditor.getByRole('button', {
      name: dictionary.notifiers.advancedSettings,
      exact: true,
    }),
  ).toHaveAttribute('aria-expanded', 'true');

  expect(
    boundary.requests.filter(
      (request) => request.path === '/notifier-tests' || request.method === 'PUT',
    ),
  ).toHaveLength(0);
});
