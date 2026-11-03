import { expect, type Locator } from '@playwright/test';

const MINIMUM_TEXT_CONTRAST = 4.5;
const WHITE = 'rgb(255, 255, 255)';

const getRelativeLuminance = (color: string) => {
  const [red, green, blue] = color
    .match(/[\d.]+/g)!
    .slice(0, 3)
    .map((channel) => {
      const value = Number(channel) / 255;

      return value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
    });

  return 0.2126 * red + 0.7152 * green + 0.0722 * blue;
};

const getContrast = ({
  foregroundColor,
  backgroundColor,
}: {
  foregroundColor: string;
  backgroundColor: string;
}) => {
  const [lighter, darker] = [
    getRelativeLuminance(foregroundColor),
    getRelativeLuminance(backgroundColor),
  ].sort((first, second) => second - first);

  return (lighter + 0.05) / (darker + 0.05);
};

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

  await expect
    .poll(async () =>
      getContrast({
        foregroundColor: WHITE,
        backgroundColor: await entry.evaluate(
          (element) => getComputedStyle(element).backgroundColor,
        ),
      }),
    )
    .toBeGreaterThanOrEqual(MINIMUM_TEXT_CONTRAST);

  if (!label) return;

  const backgroundColor = await entry.evaluate(
    (element) => getComputedStyle(element).backgroundColor,
  );
  const labelColor = await label.evaluate((element) => getComputedStyle(element).color);

  expect(getContrast({ foregroundColor: labelColor, backgroundColor })).toBeGreaterThanOrEqual(
    MINIMUM_TEXT_CONTRAST,
  );
};
