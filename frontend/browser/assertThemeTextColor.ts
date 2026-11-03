import { expect, type Locator } from '@playwright/test';

export const assertThemeTextColor = async ({
  text,
  role,
}: {
  text: Locator;
  role: '--app-muted' | '--app-danger';
}) => {
  const themeColor = await text.evaluate((element, role) => {
    const colorProbe = document.createElement('span');
    colorProbe.style.color = `var(${role})`;
    element.appendChild(colorProbe);

    const color = getComputedStyle(colorProbe).color;

    colorProbe.remove();

    return color;
  }, role);

  await expect(text).toHaveCSS('color', themeColor);
};
