import { expect, test } from '@playwright/test';

import { openSignIn, signIn } from './authentication';

test('retains saved storage and isolates workspaces after reloading the real API application', async ({
  page,
}) => {
  test.skip(
    !!process.env.DATABASUS_BROWSER_PROBE,
    'Supervisor probes exercise the dedicated sign-in scenario',
  );

  await openSignIn(page);
  await signIn(page);

  const workspaceName = `Browser archive ${crypto.randomUUID()}`;
  const otherWorkspaceName = `Browser isolated ${crypto.randomUUID()}`;
  const storageName = `Local archive ${crypto.randomUUID()}`;
  const createWorkspace = async (name: string) => {
    await page
      .getByRole('banner')
      .getByRole('button', { name: /Create workspace|Create new workspace/ })
      .click();

    const dialog = page.getByRole('dialog', { name: 'Create workspace', exact: true });
    await dialog.getByRole('textbox', { name: 'Workspace name', exact: true }).fill(name);
    await dialog.getByRole('button', { name: 'Create workspace', exact: true }).click();

    await expect(dialog).not.toBeVisible();
    await expect(
      page
        .locator('.ant-select')
        .filter({ has: page.getByRole('combobox', { name: 'Selected workspace' }) }),
    ).toContainText(name);
  };

  await createWorkspace(workspaceName);
  const workspaceId = await page.evaluate(() => localStorage.getItem('selected_workspace_id'));

  expect(workspaceId).toBeTruthy();

  await page.getByRole('button', { name: 'Storages', exact: true }).click();
  await page.getByRole('button', { name: 'Add storage', exact: true }).click();
  const storageDialog = page.getByRole('dialog', { name: 'Add storage', exact: true });
  await storageDialog.getByRole('textbox', { name: 'Name', exact: true }).fill(storageName);
  await storageDialog.getByRole('button', { name: 'Test connection', exact: true }).click();
  await storageDialog.getByRole('button', { name: 'Save', exact: true }).click();

  await expect(storageDialog).not.toBeVisible();
  await expect(page.getByRole('button', { name: storageName, exact: true })).toBeVisible();

  const token = await page.evaluate(() => localStorage.getItem('databasus_user_token'));
  const headers = { Authorization: `Bearer ${token}` };
  const storedList = await page.request.get(
    `/api/v1/storages?workspaceId=${workspaceId}&limit=100`,
    { headers },
  );

  expect(storedList.ok()).toBe(true);

  const storage = (await storedList.json()).items.find(
    (item: { name: string }) => item.name === storageName,
  );

  expect(storage).toMatchObject({ workspaceId, name: storageName, settings: { type: 'LOCAL' } });

  const storedDetail = await page.request.get(`/api/v1/storages/${storage.id}`, { headers });

  expect(storedDetail.ok()).toBe(true);
  expect(await storedDetail.json()).toMatchObject({ id: storage.id, name: storageName });

  await page.reload();
  await page.getByRole('button', { name: 'Storages', exact: true }).click();

  await expect(page.getByRole('button', { name: storageName, exact: true })).toHaveAttribute(
    'aria-pressed',
    'true',
  );
  expect(await page.evaluate(() => localStorage.getItem('selected_workspace_id'))).toBe(
    workspaceId,
  );

  await createWorkspace(otherWorkspaceName);
  const otherWorkspaceId = await page.evaluate(() => localStorage.getItem('selected_workspace_id'));

  expect(otherWorkspaceId).not.toBe(workspaceId);

  await page.getByRole('button', { name: 'Storages', exact: true }).click();

  await expect(page.getByText('No storages in this workspace yet')).toBeVisible();
  await expect(page.getByRole('button', { name: storageName, exact: true })).toHaveCount(0);

  const isolatedList = await page.request.get(
    `/api/v1/storages?workspaceId=${otherWorkspaceId}&limit=100`,
    { headers },
  );

  expect(isolatedList.ok()).toBe(true);
  expect((await isolatedList.json()).items).toEqual([]);

  await page.reload();

  await expect(
    page
      .locator('.ant-select')
      .filter({ has: page.getByRole('combobox', { name: 'Selected workspace' }) }),
  ).toContainText(otherWorkspaceName);
  expect(await page.evaluate(() => localStorage.getItem('selected_workspace_id'))).toBe(
    otherWorkspaceId,
  );

  await page.getByRole('button', { name: 'Storages', exact: true }).click();

  await expect(page.getByText('No storages in this workspace yet')).toBeVisible();

  await page.getByRole('combobox', { name: 'Selected workspace' }).focus();
  await page.getByRole('combobox', { name: 'Selected workspace' }).press('ArrowDown');
  await page.getByTitle(workspaceName, { exact: true }).click();

  await expect(page.getByRole('button', { name: storageName, exact: true })).toHaveAttribute(
    'aria-pressed',
    'true',
  );
});
