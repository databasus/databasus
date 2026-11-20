import { expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

for (const theme of ['light', 'dark']) {
  test(`focuses Wait and cancels cleanup abandonment with the keyboard in ${theme} theme`, async ({
    page,
  }) => {
    const storage: components['schemas']['Storage'] = {
      id: 'cleanup-storage',
      workspaceId: workspace.id,
      name: 'Cleanup archive',
      settings: { type: 'LOCAL' },
    };
    const deletions: string[] = [];
    await page.route(/\/api\/v1\/storages(?:\?.*)?$/, (route) =>
      route.fulfill({ json: { items: [storage] } }),
    );
    await page.route(/\/api\/v1\/storages\/cleanup-storage(?:\?.*)?$/, (route) => {
      if (route.request().method() === 'DELETE') {
        deletions.push(route.request().url());
        return route.fulfill({
          status: 409,
          contentType: 'application/problem+json',
          json: {
            type: 'about:blank',
            title: 'Conflict',
            status: 409,
            detail: 'Cleanup remains pending',
            code: 'cleanup_pending',
          },
        });
      }

      return route.fulfill({ json: storage });
    });
    await page.setViewportSize({ width: 320, height: 900 });
    await openAccount({ page, theme });
    await openWorkspaceScreen({ page, locale: 'en', tab: 'storages' });
    await page.getByRole('button', { name: /^Cleanup archive/ }).click();
    await page.getByRole('button', { name: 'Remove', exact: true }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Remove', exact: true }).click();

    const warning = page.getByRole('dialog', { name: 'Cleanup is still running' });
    await expect(warning.getByRole('button', { name: 'Wait', exact: true })).toBeFocused();

    await enlargeText(warning);
    await assertEnlargedTextReflow(page);

    await assertAccountReflow(warning);
    await assertEnlargedTextReflow(page);

    await page.keyboard.press('Escape');

    await expect(warning).toHaveCount(0);
    expect(deletions).toHaveLength(1);
    expect(new URL(deletions[0]).searchParams.has('shouldAbandonCleanup')).toBe(false);
  });
}
