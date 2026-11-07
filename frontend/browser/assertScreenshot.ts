import { expect, type Locator, type Page } from '@playwright/test';

export const assertScreenshot = async (surface: Page | Locator, name: string) => {
  const page = 'page' in surface ? surface.page() : surface;

  await page.evaluate(async () => {
    await document.fonts.ready;
    await Promise.allSettled(
      document
        .getAnimations()
        .filter((animation) => {
          const timing = animation.effect?.getComputedTiming();
          return timing && timing.iterations !== Infinity;
        })
        .map((animation) => animation.finished),
    );
  });
  await expect
    .poll(() =>
      page.evaluate(() =>
        Array.from(document.querySelectorAll<HTMLElement>('[role="dialog"]'))
          .filter((dialog) => dialog.getClientRects().length)
          .every((dialog) => {
            for (
              let element: HTMLElement | null = dialog;
              element;
              element = element.parentElement
            ) {
              if (Number(getComputedStyle(element).opacity) < 1) return false;
            }
            return true;
          }),
      ),
    )
    .toBe(true);

  await expect.soft(surface).toHaveScreenshot(name, { animations: 'disabled', maxDiffPixels: 0 });
};
