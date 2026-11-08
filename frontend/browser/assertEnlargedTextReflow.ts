import { expect, type Page } from '@playwright/test';

import { assertAccountReflow } from './assertAccountReflow';
import { enlargeText } from './enlargeText';

export const assertEnlargedTextReflow = async (page: Page) => {
  if (page.viewportSize()!.width !== 320) return;

  const listboxes = page.getByRole('listbox').filter({ visible: true });
  const dialogs = page.getByRole('dialog').filter({ visible: true });
  const menus = page.getByRole('menu').filter({ visible: true });
  const main = page.getByRole('main');
  const region = (await listboxes.count())
    ? listboxes.last()
    : (await dialogs.count())
      ? dialogs.last()
      : (await menus.count())
        ? menus.last()
        : (await main.count())
          ? main
          : page.locator('body');
  await expect
    .poll(() =>
      region.evaluate((root) => {
        for (let element: Element | null = root; element; element = element.parentElement)
          if (Number(getComputedStyle(element).opacity) < 1) return false;
        return true;
      }),
    )
    .toBe(true);
  const previous = await region.evaluateHandle((root) =>
    [root as HTMLElement, ...root.querySelectorAll<HTMLElement>('*')].map((element) => ({
      element,
      fontSize: element.style.fontSize,
      originalFontSize: element.dataset.originalFontSize,
      textScale: element.dataset.textScale,
    })),
  );

  try {
    await enlargeText(region);
    await expect
      .poll(() =>
        region.evaluate((element) => {
          const bounds = element.getBoundingClientRect();
          return bounds.left >= -1 && bounds.right <= innerWidth + 1;
        }),
      )
      .toBe(true);
    await assertAccountReflow(region);
  } finally {
    await previous.evaluate((entries) => {
      for (const { element, fontSize, originalFontSize, textScale } of entries) {
        element.style.fontSize = fontSize;
        if (originalFontSize === undefined) delete element.dataset.originalFontSize;
        else element.dataset.originalFontSize = originalFontSize;
        if (textScale === undefined) delete element.dataset.textScale;
        else element.dataset.textScale = textScale;
      }
    });
    await previous.dispose();
  }
};
