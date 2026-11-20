import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { applicationFixtures } from '../applicationFixtures';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';

for (const authMethods of [
  ['PRIVATE_KEY'],
  ['PASSWORD', 'PRIVATE_KEY'],
] as components['schemas']['SftpAuthMethod'][][]) {
  test.describe(`SFTP ${authMethods.join(' and ')}`, () => {
    const storage: components['schemas']['Storage'] = {
      id: 'sftp-id',
      workspaceId: 'workspace-id',
      name: 'SFTP archive',
      settings: {
        type: 'SFTP',
        host: 'sftp.example.com',
        port: 22,
        username: 'backup',
        authMethods,
        isSkipHostKeyVerify: false,
      },
    };
    test.use({
      httpFixtures: [
        [
          ...applicationFixtures().filter((fixture) => fixture.path !== '/storages'),
          apiResponse({
            method: 'get',
            path: '/storages',
            status: 200,
            body: { items: [storage] },
          }),
          apiResponse({ method: 'get', path: '/storages/{storageId}', status: 200, body: storage }),
          apiResponse({ method: 'put', path: '/storages/{storageId}', status: 200, body: storage }),
          apiResponse({
            method: 'post',
            path: '/storage-tests',
            status: 200,
            body: { isConnected: true },
          }),
        ],
        { scope: 'test' },
      ],
    });

    test('reads saved methods, renames without secrets and switches explicitly through accessible narrow fields', async ({
      page,
      boundary,
    }) => {
      await openAccount({ page });
      await page.getByRole('link', { name: 'Storages', exact: true }).click();
      await expect(page.getByText('sftp.example.com')).toBeVisible();
      await expect(page.getByText('Private key', { exact: true })).toBeVisible();
      if (authMethods.includes('PASSWORD'))
        await expect(page.getByText('Password', { exact: true })).toBeVisible();
      else await expect(page.getByText('Password', { exact: true })).toHaveCount(0);

      await page.getByRole('button', { name: 'Edit', exact: true }).first().click();
      await page.getByPlaceholder('Enter name...').fill('Renamed SFTP archive');
      await page.getByRole('button', { name: 'Save', exact: true }).click();
      await expect
        .poll(() => boundary.requests.filter((request) => request.method === 'PUT').length)
        .toBe(1);
      const rename = boundary.requests.find((request) => request.method === 'PUT');
      expect(rename?.body).toEqual({ name: 'Renamed SFTP archive', settings: storage.settings });

      await page.getByRole('button', { name: 'Edit', exact: true }).last().click();
      await expect(page.getByLabel('Private key', { exact: true })).toHaveValue('');
      await expect(page.getByText('A private key is saved. Leave blank to keep it.')).toBeVisible();
      if (authMethods.includes('PASSWORD'))
        await expect(page.getByLabel('Password', { exact: true })).toHaveValue('');

      await page.setViewportSize({ width: 320, height: 1000 });
      const form = page.getByRole('region', { name: 'Storage settings', exact: true });
      const accessibility = await new AxeBuilder({ page })
        .disableRules(['color-contrast'])
        .include('section[aria-label="Storage settings"]')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();
      expect(accessibility.violations).toEqual([]);
      const geometry = await form.evaluate((element) => ({
        width: element.scrollWidth,
        client: element.clientWidth,
        right: element.getBoundingClientRect().right,
        viewport: innerWidth,
      }));
      expect(geometry.width).toBeLessThanOrEqual(geometry.client + 1);
      expect(geometry.right).toBeLessThanOrEqual(geometry.viewport);

      const method = page.getByRole('combobox', { name: 'Auth method', exact: true });
      await method.focus();
      await expect(method).toBeFocused();
      await method.press('ArrowDown');
      await page.getByRole('option', { name: 'Password', exact: true }).click();
      await expect(page.getByLabel('Private key', { exact: true })).toHaveCount(0);
      if (!authMethods.includes('PASSWORD')) {
        await expect(
          page.getByRole('button', { name: 'Test connection', exact: true }),
        ).toBeDisabled();
        await expect(page.getByText('Enter a password for this method.')).toBeVisible();
      }

      await page.getByLabel('Password', { exact: true }).fill('replacement-password');
      await page.getByRole('button', { name: 'Test connection', exact: true }).click();
      await page.getByRole('button', { name: 'Save', exact: true }).click();
      await expect
        .poll(() => boundary.requests.filter((request) => request.method === 'PUT').length)
        .toBe(2);
      const switchRequest = boundary.requests.filter((request) => request.method === 'PUT')[1];
      expect(switchRequest.body).toMatchObject({
        settings: { authMethods: ['PASSWORD'], password: 'replacement-password' },
      });
      expect(
        (switchRequest.body as components['schemas']['StorageUpdate']).settings,
      ).not.toHaveProperty('privateKey');

      const testRequest = boundary.requests.find((request) => request.path === '/storage-tests');
      expect(testRequest?.body).toMatchObject({
        storageId: storage.id,
        settings: { authMethods: ['PASSWORD'], password: 'replacement-password' },
      });
    });
  });
}
