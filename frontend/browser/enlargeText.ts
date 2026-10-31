import { expect, type Locator } from '@playwright/test';

export const enlargeText = async (region: Locator) => {
  const probe = region.locator('input, button, label').first();
  const before = await probe.evaluate((element) => parseFloat(getComputedStyle(element).fontSize));
  await region.evaluate((root) => {
    const sizes = Array.from(root.querySelectorAll<HTMLElement>('*')).map((element) => ({
      element,
      size: parseFloat(getComputedStyle(element).fontSize),
    }));
    for (const { element, size } of sizes) element.style.fontSize = `${size * 2}px`;
  });
  await expect(probe).toHaveCSS('font-size', `${before * 2}px`);
};
