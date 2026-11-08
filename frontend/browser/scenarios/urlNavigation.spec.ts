import { expect } from '@playwright/test';

import { workspace } from '../applicationFixtures';
import { test } from '../browserTest';
import { database, databaseFixtures } from '../databaseFixtures';
import { openAccount } from '../openAccount';

const databasePath = `/workspaces/${workspace.id}/databases/${database.id}`;

test.use({ httpFixtures: [databaseFixtures(), { scope: 'test' }] });

test('restores the selected database tab after a reload', async ({ page }) => {
  await openAccount({ page });
  await page.goto(`${databasePath}/config`);
  await expect(page.getByRole('tab', { name: 'Config', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  );

  await page.reload();

  await expect(page.getByRole('tab', { name: 'Config', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(page).toHaveURL(`${databasePath}/config`);
});

test('returns to the previous tab with the back button', async ({ page }) => {
  await openAccount({ page });
  await page.goto(`${databasePath}/backups`);
  await page.getByRole('tab', { name: 'Config', exact: true }).click();
  await expect(page).toHaveURL(`${databasePath}/config`);

  await page.goBack();

  await expect(page.getByRole('tab', { name: 'Backups', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(page).toHaveURL(`${databasePath}/backups`);
  await page.goForward();
  await expect(page).toHaveURL(`${databasePath}/config`);
  await expect(page.getByRole('tab', { name: 'Config', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  );
});

test('restores the selected application section after a reload', async ({ page }) => {
  await openAccount({ page });
  await page.getByRole('link', { name: 'Storages', exact: true }).click();
  await expect(page).toHaveURL(`/workspaces/${workspace.id}/storages`);

  await page.reload();

  await expect(page.getByRole('link', { name: 'Storages', exact: true })).toHaveAttribute(
    'aria-current',
    'page',
  );
  await expect(page).toHaveURL(`/workspaces/${workspace.id}/storages`);
});

test('opens mobile details directly and returns to the database list', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await openAccount({ page });
  await page.goto(`${databasePath}/config`);
  await expect(page.getByRole('tab', { name: 'Config', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  );

  await page.getByRole('button', { name: '← Back to databases', exact: true }).click();

  await expect(page).toHaveURL(`/workspaces/${workspace.id}/databases`);
  await expect(page.getByRole('link', { name: /^Orders/ })).toBeVisible();
  await expect(page.getByRole('tab')).toHaveCount(0);
});

test('replaces an unknown database tab with its backups address', async ({ page }) => {
  await openAccount({ page });
  await page.goto(`${databasePath}/unknown`);

  await expect(page).toHaveURL(`${databasePath}/backups`);
  await expect(page.getByRole('tab', { name: 'Backups', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  );
});
