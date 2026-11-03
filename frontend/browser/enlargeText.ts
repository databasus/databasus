import { expect, type Locator } from '@playwright/test';

export const enlargeText = async (region: Locator) => {
  const probe = region.locator('input, button, label').first();
  const before = await probe.evaluate((element) => parseFloat(getComputedStyle(element).fontSize));
  await region.evaluate(async (root) => {
    const sizes = Array.from(root.querySelectorAll<HTMLElement>('*')).map((element) => ({
      element,
      size: parseFloat(getComputedStyle(element).fontSize),
    }));
    for (const { element, size } of sizes) element.style.fontSize = `${size * 2}px`;

    // AntD controls animate font-size, so the enlarged layout exists only after these transitions.
    await Promise.allSettled(
      root
        .getAnimations({ subtree: true })
        .filter((animation) => animation instanceof CSSTransition)
        .map((animation) => animation.finished),
    );
  });
  await expect(probe).toHaveCSS('font-size', `${before * 2}px`);

  // AntD animates font-size on some controls, such as a select's value, and a layout measured
  // during that transition sees text that has not reached its final size.
  await region.evaluate(async (root) => {
    const fontTransitions = root
      .getAnimations({ subtree: true })
      .filter(
        (animation) =>
          animation instanceof CSSTransition && animation.transitionProperty === 'font-size',
      );
    await Promise.allSettled(fontTransitions.map((transition) => transition.finished));
  });
};
