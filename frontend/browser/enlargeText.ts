import { expect, type Locator } from '@playwright/test';

export const enlargeText = async (region: Locator) => {
  const control = region.locator('input, button, label').first();
  const probe = (await control.count()) ? control : region;
  const before = await region.evaluate(async (root) => {
    const elementsWithTransitions = Array.from(document.querySelectorAll<HTMLElement>('*')).map(
      (element) => ({
        element,
        transition: element.style.getPropertyValue('transition'),
        priority: element.style.getPropertyPriority('transition'),
      }),
    );
    for (const { element } of elementsWithTransitions)
      element.style.setProperty('transition', 'none', 'important');

    let probeSize: number;
    try {
      const previousSizes = Array.from(
        document.querySelectorAll<HTMLElement>('[data-original-font-size]'),
      ).map((element) => ({ element, size: element.style.fontSize }));
      for (const { element } of previousSizes)
        element.style.fontSize = `${element.dataset.originalFontSize}px`;

      const probe = root.querySelector('input, button, label') ?? root;
      probeSize = parseFloat(getComputedStyle(probe).fontSize);
      (root as HTMLElement).dataset.textScale = '2';
      const elements = Array.from(root.querySelectorAll<HTMLElement>('*'));
      if (!root.querySelector('input, button, label')) elements.unshift(root as HTMLElement);
      const sizes = elements.map((element) => ({
        element,
        size: parseFloat(getComputedStyle(element).fontSize),
      }));
      for (const { element, size } of previousSizes) element.style.fontSize = size;

      for (const { element, size } of sizes) {
        element.dataset.originalFontSize = String(size);
        element.style.fontSize = `${size * 2}px`;
      }

      root.getBoundingClientRect();
    } finally {
      for (const { element, transition, priority } of elementsWithTransitions) {
        if (transition) element.style.setProperty('transition', transition, priority);
        else element.style.removeProperty('transition');
      }
    }

    // The enlarged geometry is final only after any font-size transition finishes.
    await Promise.allSettled(
      root
        .getAnimations({ subtree: true })
        .filter((animation) => animation instanceof CSSTransition)
        .map((animation) => animation.finished),
    );

    return probeSize;
  });
  await expect(probe).toHaveCSS('font-size', `${before * 2}px`);

  // Wait for font-size transitions triggered by descendants after the probe reached its size.
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
