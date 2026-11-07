import { expect, type Page } from '@playwright/test';
import dayjs from 'dayjs';
import 'dayjs/locale/fr.js';

import { DICTIONARIES, type Locale } from '../src/shared/i18n/dictionaries';
import { assertEnlargedTextReflow } from './assertEnlargedTextReflow';

export const verifyBackupFilters = async ({
  page,
  locale,
  backupKind,
}: {
  page: Page;
  locale: Locale;
  backupKind: 'logical' | 'physical';
}) => {
  const dictionary = DICTIONARIES[locale];
  const endpoint = `/${backupKind}-backups`;
  const waitForFilter = (query: { status?: string; type?: string; createdBefore?: string }) =>
    page.waitForRequest((request) => {
      const url = new URL(request.url());
      return (
        request.method() === 'GET' &&
        url.pathname.endsWith(endpoint) &&
        ['status', 'type', 'createdBefore'].every(
          (key) => url.searchParams.get(key) === (query[key as keyof typeof query] ?? null),
        )
      );
    });
  const status = page.getByRole('combobox', { name: dictionary.common.fields.status, exact: true });

  await status.focus();
  await status.press('ArrowDown');
  await assertEnlargedTextReflow(page);
  const statusRequest = waitForFilter({ status: 'COMPLETED' });
  await page
    .getByRole('option', {
      name: dictionary.status[backupKind === 'logical' ? 'logicalBackup' : 'physicalBackup']
        .completed,
      exact: true,
    })
    .click();
  await statusRequest;
  await page.keyboard.press('Escape');

  await expect(status).toHaveAttribute('aria-expanded', 'false');
  await expect(status).toBeFocused();

  if (backupKind === 'physical') {
    const type = page.getByRole('combobox', { name: dictionary.common.fields.type, exact: true });
    await type.click();
    const typeRequest = waitForFilter({ status: 'COMPLETED', type: 'INCREMENTAL' });
    await page
      .getByRole('option', { name: dictionary.backups.physical.types.incremental, exact: true })
      .click();
    await typeRequest;
    await page.keyboard.press('Escape');
  }

  const date = page.getByRole('textbox', { name: dictionary.backups.filters.before, exact: true });
  const day = page.getByRole('button', {
    name: dayjs('2026-10-17').locale(locale).format('dddd, D MMMM YYYY'),
    exact: true,
  });
  await date.click();
  await expect(day).toBeVisible();
  await assertEnlargedTextReflow(page);
  await day.focus();
  await page.keyboard.press('Escape');

  await expect(day).toBeHidden();
  await expect(date).toBeFocused();

  await date.click();
  const dateRequest = waitForFilter({
    status: 'COMPLETED',
    type: backupKind === 'physical' ? 'INCREMENTAL' : undefined,
    createdBefore: '2026-10-17T00:00:00.000Z',
  });
  await day.click();
  await dateRequest;

  await expect(date).not.toHaveValue('');

  const clearDate = waitForFilter({
    status: 'COMPLETED',
    type: backupKind === 'physical' ? 'INCREMENTAL' : undefined,
  });
  await page
    .getByRole('button', { name: dictionary.common.actions.clear, exact: true })
    .last()
    .click();
  await clearDate;

  await expect(date).toHaveValue('');
  await expect(
    page.getByRole('button', { name: dictionary.common.actions.clear, exact: true }),
  ).toHaveCount(backupKind === 'physical' ? 2 : 1);

  const clearStatus = waitForFilter({
    type: backupKind === 'physical' ? 'INCREMENTAL' : undefined,
  });
  await page
    .getByRole('button', { name: dictionary.common.actions.clear, exact: true })
    .last()
    .click();
  await clearStatus;
  await expect(
    page.getByRole('button', { name: dictionary.common.actions.clear, exact: true }),
  ).toHaveCount(backupKind === 'physical' ? 1 : 0);
  if (backupKind === 'physical') {
    const clearType = waitForFilter({});
    await page.getByRole('button', { name: dictionary.common.actions.clear, exact: true }).click();
    await clearType;
  }

  await expect(
    page.getByRole('button', { name: dictionary.common.actions.clear, exact: true }),
  ).toHaveCount(0);
};
