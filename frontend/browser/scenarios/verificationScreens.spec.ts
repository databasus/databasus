import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES, type Locale } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { assertScreenshot } from '../assertScreenshot';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { visualContexts } from '../visualContexts';

const dictionary = DICTIONARIES.en;
const runsCopy = dictionary.verification.runs;
const failureOutput =
  'pg_restore: error: type geometry does not exist\n' +
  'long_object_name_'.repeat(1800) +
  '\n[... diagnostic output truncated ...]\n' +
  'long_object_name_'.repeat(1800) +
  '\npg_restore: warning: errors ignored on restore: 1';
const REVEALED_TOKEN = 'a'.repeat(64);

// More presses than the drawer has controls, so the focus goes around at least once.
const FOCUS_TRAP_TAB_PRESSES = 12;

const database: components['schemas']['Database'] = {
  id: 'orders-id',
  workspaceId: workspace.id,
  name: 'Orders',
  notifierIds: [],
  settings: {
    type: 'POSTGRES_LOGICAL',
    host: 'postgres.example.com',
    port: 5432,
    username: 'backup',
    database: 'orders',
    sslMode: 'disable',
    sshTunnel: { isEnabled: false },
    includeSchemas: [],
    excludeTables: [],
    excludeTableData: [],
    cpuCount: 1,
    isSkipUserMappings: false,
  },
};

const failedVerification: components['schemas']['Verification'] = {
  id: 'failed-verification',
  databaseId: database.id,
  backupId: 'backup-id',
  trigger: 'MANUAL',
  status: 'FAILED',
  attemptCount: 2,
  createdAt: '2026-10-18T10:00:00Z',
  startedAt: '2026-10-18T10:01:00Z',
  finishedAt: '2026-10-18T11:03:00Z',
  restoreDurationMs: 3_720_000,
  failMessage: 'The restore stopped with exit code 1.',
};

const agent: components['schemas']['VerificationAgent'] = {
  id: 'agent-id',
  name: 'Restore host',
  maxCpu: 2,
  maxRamMb: 2048,
  maxDiskGb: 20,
  maxConcurrentJobs: 1,
  createdAt: '2026-10-18T10:00:00Z',
};

test.use({
  httpFixtures: [
    [
      ...accountFixtures().filter(
        (fixture) =>
          !['/databases', '/backup-configs', '/storages', '/verification-agents'].includes(
            fixture.path,
          ),
      ),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/databases/{databaseId}', status: 200, body: database }),
      apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [] } }),
      apiResponse({ method: 'get', path: '/backup-configs', status: 200, body: { items: [] } }),
      apiResponse({
        method: 'get',
        path: '/logical-backups',
        status: 200,
        body: { totalUsageMb: 0, items: [] },
      }),
      apiResponse({
        method: 'get',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: {
          cleanup: { pendingFileCount: 0, failedFileCount: 0 },
          databaseId: database.id,
          storageId: null,
          isBackupsEnabled: false,
          encryption: 'NONE',
          settings: {
            type: 'LOGICAL',
            schedule: { type: 'DAILY', timeOfDay: '04:00' },
            retention: {
              policy: 'COUNT',
              count: 10,
              period: 'MONTH',
              gfsHours: 0,
              gfsDays: 0,
              gfsWeeks: 0,
              gfsMonths: 0,
              gfsYears: 0,
            },
            isRetryIfFailed: false,
            maxFailedTriesCount: 1,
            notificationTriggers: [],
          },
        },
      }),
      apiResponse({
        method: 'get',
        path: '/healthcheck-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isHealthcheckEnabled: false,
          isNotificationEnabled: false,
          intervalMinutes: 1,
          failedAttemptsBeforeUnavailable: 3,
          attemptRetentionDays: 7,
        },
      }),
      apiResponse({
        method: 'get',
        path: '/healthcheck-attempts',
        status: 200,
        body: { items: [] },
      }),
      apiResponse({
        method: 'get',
        path: '/verification-agent-counts',
        status: 200,
        body: { agentCount: 1 },
      }),
      apiResponse({
        method: 'get',
        path: '/verifications',
        status: 200,
        body: { items: [failedVerification] },
      }),
      apiResponse({
        method: 'get',
        path: '/verifications/{verificationId}',
        status: 200,
        body: {
          ...failedVerification,
          failureDiagnostic: { output: failureOutput, isTruncated: true },
          tableStats: [{ schemaName: 'public', name: 'orders', rowCount: 1200 }],
        },
      }),
      apiResponse({
        method: 'get',
        path: '/verification-agents',
        status: 200,
        body: { items: [agent] },
      }),
      apiResponse({
        method: 'post',
        path: '/verification-agents/{agentId}/rotate-token',
        status: 200,
        body: { token: REVEALED_TOKEN },
      }),
    ],
    { scope: 'test' },
  ],
});

const isFocusInside = (container: Locator) =>
  container.evaluate((element) => element.contains(document.activeElement));

// A dialog takes the focus at the end of its opening animation, and its keys work only after that.
const waitForFocusInside = (container: Locator) =>
  expect.poll(() => isFocusInside(container)).toBe(true);

const serveAgentInEveryStatus = (page: Page) =>
  page.route(/\/api\/v1\/verification-agents(\?.*)?$/, (route) => {
    const seenAgo = (minutes: number) =>
      new Date(Date.parse('2026-10-18T12:00:00Z') - minutes * 60_000).toISOString();

    return route.fulfill({
      json: {
        items: [
          agent,
          { ...agent, id: 'online-agent', name: 'Online host', lastSeenAt: seenAgo(0) },
          { ...agent, id: 'stale-agent', name: 'Stale host', lastSeenAt: seenAgo(3) },
          { ...agent, id: 'offline-agent', name: 'Offline host', lastSeenAt: seenAgo(30) },
        ],
      },
    });
  });

const openVerificationList = async (page: Page, theme: string, locale: Locale = 'en') => {
  await openAccount({ page, locale, theme });
  await openWorkspaceScreen({ page, locale, tab: 'databases' });
  if (page.viewportSize()!.width < 768)
    await page.getByRole('link', { name: 'Orders', exact: true }).click();
  await page
    .getByRole('main')
    .getByText(DICTIONARIES[locale].databases.tabs.verifications, { exact: true })
    .click();
};

for (const theme of ['light', 'dark']) {
  test(`keeps focus inside the verification details and returns it to the row action in the ${theme} theme`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await openVerificationList(page, theme);

    const detailsAction = page
      .getByRole('table')
      .getByRole('button', { name: runsCopy.actions.viewDetails });
    await detailsAction.focus();
    await page.keyboard.press('Enter');

    const drawer = page.getByRole('dialog');
    await expect(drawer.getByText(failedVerification.failMessage!)).toBeVisible();
    await expect(drawer.getByText('1h 2m 0s')).toBeVisible();

    // Focus remains within the open drawer while its details are inspected.
    const drawerRoot = page.getByRole('dialog');
    await waitForFocusInside(drawerRoot);

    const disclosure = drawer.getByRole('button', {
      name: new RegExp(runsCopy.details.diagnostic.title),
    });
    await expect(disclosure).toHaveAttribute('aria-expanded', 'false');
    await disclosure.focus();
    await page.keyboard.press('Enter');
    await expect(disclosure).toHaveAttribute('aria-expanded', 'true');
    await expect(drawer.locator('pre')).toHaveText(failureOutput);
    await expect(drawer.getByText(runsCopy.details.diagnostic.truncated)).toBeVisible();
    await page.keyboard.press('Enter');
    await expect(disclosure).toHaveAttribute('aria-expanded', 'false');

    const accessibility = await new AxeBuilder({ page })
      .disableRules(['color-contrast'])
      .include('[role="dialog"]')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(accessibility.violations).toEqual([]);

    for (let tabPress = 0; tabPress < FOCUS_TRAP_TAB_PRESSES; tabPress++) {
      await page.keyboard.press('Tab');

      expect(await isFocusInside(drawerRoot)).toBe(true);
    }
    await assertEnlargedTextReflow(page);

    await page.keyboard.press('Escape');

    await expect(drawer).toBeHidden();
    await expect(detailsAction).toBeFocused();
  });
}

for (const theme of ['light', 'dark']) {
  test(`keeps the agent table accessible and returns focus to its action when the rotate dialog closes in the ${theme} theme`, async ({
    page,
  }) => {
    await serveAgentInEveryStatus(page);
    await page.setViewportSize({ width: 1280, height: 900 });
    await openAccount({ page, locale: 'en', theme });
    await openAccountTab({ page, locale: 'en', tab: 'databasusSettings' });

    const table = page.getByRole('table');
    for (const status of Object.values(dictionary.status.agent)) {
      await expect(table.getByText(status, { exact: true })).toBeVisible();
    }

    const rotateAction = table
      .getByRole('button', { name: dictionary.verification.agents.actions.rotateToken })
      .first();

    const accessibility = await new AxeBuilder({ page })
      .disableRules(['color-contrast'])
      .include('table')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(accessibility.violations).toEqual([]);

    await rotateAction.focus();
    await page.keyboard.press('Enter');

    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText(agent.name)).toBeVisible();
    await waitForFocusInside(dialog);
    await assertEnlargedTextReflow(page);

    await page.keyboard.press('Escape');

    await expect(dialog).toBeHidden();
    await expect(rotateAction).toBeFocused();
  });
}

for (const { width, theme } of [
  { width: 320, theme: 'light' },
  { width: 1280, theme: 'light' },
  { width: 1280, theme: 'dark' },
] as const) {
  test(`shows the revealed token and its commands in French at ${width}px in ${theme} theme`, async ({
    page,
  }) => {
    const frenchCopy = DICTIONARIES.fr.verification.agents;

    await page.setViewportSize({ width, height: 800 });
    await openAccount({ page, locale: 'fr', theme });
    await openAccountTab({ page, locale: 'fr', tab: 'databasusSettings' });

    await page.getByRole('button', { name: frenchCopy.actions.rotateToken }).click();
    await page
      .getByRole('dialog')
      .getByRole('button', { name: frenchCopy.actions.rotateToken })
      .click();

    const dialog = page.getByRole('dialog', { name: frenchCopy.token.title });
    await expect(dialog.locator('pre').first()).toHaveText(REVEALED_TOKEN);
    await expect(dialog.getByText(`--token=${REVEALED_TOKEN}`)).toBeVisible();
    await assertEnlargedTextReflow(page);

    await assertAccountReflow(dialog);
    await waitForFocusInside(dialog);
    await expect(dialog).toHaveCSS('opacity', '1');

    const accessibility = await new AxeBuilder({ page })
      .disableRules(['color-contrast'])
      .include('[role="dialog"]')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(accessibility.violations).toEqual([]);
  });
}

for (const { theme, locale } of [
  { theme: 'light', locale: 'en' },
  { theme: 'dark', locale: 'en' },
  { theme: 'light', locale: 'fr' },
] as const) {
  test(`contains long failure output inside the drawer at 320px with 200% text in the ${theme} theme and ${locale} locale`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 800 });
    const dictionary = DICTIONARIES[locale];
    const runsCopy = dictionary.verification.runs;
    await openVerificationList(page, theme, locale);
    await assertEnlargedTextReflow(page);
    await page.getByRole('button', { name: runsCopy.actions.viewDetails }).click();
    const drawer = page.getByRole('dialog');
    await drawer
      .getByRole('button', { name: new RegExp(runsCopy.details.diagnostic.title) })
      .click();
    await expect(drawer.locator('pre')).toHaveText(failureOutput);
    await enlargeText(drawer);
    await assertEnlargedTextReflow(page);

    if (theme === 'dark')
      await assertScreenshot(page, 'verification-diagnostics-dark-narrow-200.png');
    await assertAccountReflow(drawer);
    const diagnosticLayout = await drawer.locator('pre').evaluate((output) => ({
      isLocallyScrollable: output.scrollWidth > output.clientWidth,
      isInsideDrawer:
        output.getBoundingClientRect().right <=
        output.closest('[role="dialog"]')!.getBoundingClientRect().right,
      overflowX: getComputedStyle(output).overflowX,
    }));
    expect(diagnosticLayout).toEqual({
      isLocallyScrollable: true,
      isInsideDrawer: true,
      overflowX: 'auto',
    });
    await drawer
      .getByRole('button', { name: dictionary.common.actions.copy })
      .scrollIntoViewIfNeeded();
    await expect(
      drawer.getByRole('button', { name: dictionary.common.actions.copy }),
    ).toBeVisible();
  });
}

for (const context of visualContexts) {
  test(`opens agent creation, installation and anchored deletion in ${context.theme} ${context.locale} ${context.width}`, async ({
    page,
  }) => {
    const dictionary = DICTIONARIES[context.locale];
    const copy = dictionary.verification.agents;
    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context });
    await openAccountTab({ page, locale: context.locale, tab: 'databasusSettings' });
    const create = page.getByRole('button', { name: dictionary.common.actions.create });
    await expect(create).toBeVisible();
    await assertEnlargedTextReflow(page);

    if (context.width === 320) {
      const tableRegion = page.getByRole('region', { name: copy.title, exact: true });
      await tableRegion.focus();
      await Promise.all([
        tableRegion.evaluate(
          (region) =>
            new Promise<void>((resolve) =>
              region.addEventListener('scrollend', () => resolve(), { once: true }),
            ),
        ),
        tableRegion.press('ArrowRight'),
      ]);
      await expect
        .poll(() => tableRegion.evaluate((region) => region.scrollLeft))
        .toBeGreaterThan(0);
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
        320,
      );
      await tableRegion.evaluate((region) => {
        region.scrollTo({ left: 0, behavior: 'instant' });
      });
      await expect.poll(() => tableRegion.evaluate((region) => region.scrollLeft)).toBe(0);
    }
    await create.click();
    const creation = page.getByRole('dialog', { name: copy.create.title, exact: true });
    const name = creation.getByRole('textbox', {
      name: dictionary.common.fields.name,
      exact: true,
    });
    await name.fill('Remote verification host');
    await assertEnlargedTextReflow(page);

    await page.keyboard.press('Escape');
    await expect(creation).toBeHidden();
    await create.click();
    await expect(name).toHaveValue('');
    await name.focus();
    await page.keyboard.press('Escape');
    await expect(creation).toBeHidden();
    await page.getByRole('button', { name: copy.actions.viewInstallCommands, exact: true }).click();
    const installation = page.getByRole('dialog', { name: copy.install.title, exact: true });
    await expect(installation.locator('pre').first()).toContainText('verification-agent');
    await assertEnlargedTextReflow(page);

    await installation.getByText('arm64', { exact: true }).click();
    await expect(installation.locator('pre').first()).toContainText('arm64');
    await assertEnlargedTextReflow(page);

    if (context.width === 320) {
      await enlargeText(installation);

      await assertAccountReflow(installation);
    }
    await page.keyboard.press('Escape');
    await expect(installation).toBeHidden();
    await page.getByRole('button', { name: dictionary.common.actions.delete, exact: true }).click();
    await expect(page.getByText(copy.deleteConfirmation, { exact: true })).toBeVisible();
    await assertEnlargedTextReflow(page);

    const cancel = page.getByRole('button', {
      name: dictionary.common.actions.cancel,
      exact: true,
    });
    expect
      .soft(
        await cancel.evaluate((button) => {
          const bounds = button.getBoundingClientRect();
          const target = document.elementFromPoint(
            bounds.left + bounds.width / 2,
            bounds.top + bounds.height / 2,
          );
          return target === button || button.contains(target);
        }),
      )
      .toBe(true);
    await cancel.focus();
    await page.keyboard.press('Enter');
    await expect(page.getByText(copy.deleteConfirmation, { exact: true })).toBeHidden();
  });
}
