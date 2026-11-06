import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

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
      apiResponse({ method: 'get', path: '/logical-backups', status: 200, body: { items: [] } }),
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

// Agents' ages count from each answer, because a status depends on the time since the agent was seen.
const serveAgentInEveryStatus = (page: Page) =>
  page.route(/\/api\/v1\/verification-agents(\?.*)?$/, (route) => {
    const seenAgo = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString();

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

const openVerificationList = async (page: Page, theme: string) => {
  await openAccount({ page, locale: 'en', theme });
  await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });
  if (page.viewportSize()!.width < 768)
    await page.getByRole('button', { name: 'Orders', exact: true }).click();
  await page
    .getByRole('main')
    .getByText(dictionary.databases.tabs.verifications, { exact: true })
    .click();
};

for (const theme of ['light', 'dark']) {
  test(`keeps focus inside the verification details and returns it to the row action in the ${theme} theme`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await openVerificationList(page, theme);

    const detailsAction = page
      .locator('.ant-table-wrapper')
      .getByRole('button', { name: runsCopy.actions.viewDetails });
    await detailsAction.focus();
    await page.keyboard.press('Enter');

    const drawer = page.getByRole('dialog');
    await expect(drawer.getByText(failedVerification.failMessage!)).toBeVisible();
    await expect(drawer.getByText('1h 2m 0s')).toBeVisible();

    // The drawer keeps the focus on its root and sentinels, which wrap the dialog element.
    const drawerRoot = page.locator('.ant-drawer');
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
      .include('.ant-drawer-content')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(accessibility.violations).toEqual([]);

    for (let tabPress = 0; tabPress < FOCUS_TRAP_TAB_PRESSES; tabPress++) {
      await page.keyboard.press('Tab');

      expect(await isFocusInside(drawerRoot)).toBe(true);
    }

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

    const table = page.locator('.ant-table-wrapper');
    for (const status of Object.values(dictionary.status.agent)) {
      await expect(table.getByText(status, { exact: true })).toBeVisible();
    }

    const rotateAction = table
      .getByRole('button', { name: dictionary.verification.agents.actions.rotateToken })
      .first();

    const accessibility = await new AxeBuilder({ page })
      .disableRules(['color-contrast'])
      .include('.ant-table-wrapper')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(accessibility.violations).toEqual([]);

    await rotateAction.focus();
    await page.keyboard.press('Enter');

    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText(agent.name)).toBeVisible();
    await waitForFocusInside(dialog);

    await page.keyboard.press('Escape');

    await expect(dialog).toBeHidden();
    await expect(rotateAction).toBeFocused();
  });
}

test('shows the revealed token and its commands in a long locale at a narrow viewport', async ({
  page,
}) => {
  const frenchCopy = DICTIONARIES.fr.verification.agents;

  await page.setViewportSize({ width: 320, height: 800 });
  await openAccount({ page, locale: 'fr', theme: 'light' });
  await openAccountTab({ page, locale: 'fr', tab: 'databasusSettings' });

  await page.getByRole('button', { name: frenchCopy.actions.rotateToken }).click();
  await page
    .getByRole('dialog')
    .getByRole('button', { name: frenchCopy.actions.rotateToken })
    .click();

  const dialog = page.getByRole('dialog', { name: frenchCopy.token.title });
  await expect(dialog.locator('pre').first()).toHaveText(REVEALED_TOKEN);
  await expect(dialog.getByText(`--token=${REVEALED_TOKEN}`)).toBeVisible();

  await assertAccountReflow(dialog);
  await waitForFocusInside(dialog);
  await expect(dialog).toHaveCSS('opacity', '1');

  const accessibility = await new AxeBuilder({ page })
    .disableRules(['color-contrast'])
    .include('.ant-modal-content')
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();

  expect(accessibility.violations).toEqual([]);
});

for (const theme of ['light', 'dark']) {
  test(`contains long failure output inside the drawer at 320px with 200% text in the ${theme} theme`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 800 });
    await openVerificationList(page, theme);
    await page.getByRole('button', { name: runsCopy.actions.viewDetails }).click();
    const drawer = page.getByRole('dialog');
    await drawer
      .getByRole('button', { name: new RegExp(runsCopy.details.diagnostic.title) })
      .click();
    await expect(drawer.locator('pre')).toHaveText(failureOutput);
    await enlargeText(drawer);

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
