import { expect, type Locator } from '@playwright/test';

export const assertSelectedNavigationEntry = async ({
  entry,
  iconName,
  label,
}: {
  entry: Locator;
  iconName: string;
  label?: Locator;
}) => {
  await expect(entry).toHaveAttribute('aria-current', 'page');
  await expect(entry.locator('img')).toHaveAttribute('src', `/icons/menu/${iconName}-white.svg`);
  await expect(entry.locator('img')).toHaveAttribute('alt', '');

  await expect(entry).not.toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');

  if (label) await expect(label).toHaveCSS('color', 'rgb(255, 255, 255)');
};
