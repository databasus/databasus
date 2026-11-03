import { expect, type Locator } from '@playwright/test';

export const assertAccountReflow = async (region: Locator) => {
  const layout = await region.evaluate((root) => {
    const elements = Array.from(
      root.querySelectorAll<HTMLElement>(
        'a[href], label[for], input, button, .ant-form-item-explain, .ant-alert',
      ),
    ).filter(
      (element) => element.getClientRects().length && !element.closest('.ant-input-clear-icon'),
    );
    const overlaps: string[] = [];
    for (let leftIndex = 0; leftIndex < elements.length; leftIndex++)
      for (let rightIndex = leftIndex + 1; rightIndex < elements.length; rightIndex++) {
        const leftElement = elements[leftIndex],
          rightElement = elements[rightIndex];
        if (leftElement.contains(rightElement) || rightElement.contains(leftElement)) continue;
        const leftBounds = leftElement.getBoundingClientRect(),
          rightBounds = rightElement.getBoundingClientRect();
        if (
          Math.min(leftBounds.right, rightBounds.right) -
            Math.max(leftBounds.left, rightBounds.left) >
            1 &&
          Math.min(leftBounds.bottom, rightBounds.bottom) -
            Math.max(leftBounds.top, rightBounds.top) >
            1
        )
          overlaps.push(
            `${leftElement.textContent || leftElement.getAttribute('aria-label')} / ${rightElement.textContent || rightElement.getAttribute('aria-label')}`,
          );
      }

    const overflowingRegions = [root, ...root.querySelectorAll<HTMLElement>('*')]
      .filter(
        (element) =>
          element.getClientRects().length &&
          element.clientWidth > 0 &&
          !!element.textContent?.trim() &&
          // AntD ends a long select value with an ellipsis; its title and options keep the full text.
          !element.matches('input, textarea, svg, svg *, .ant-select-selection-item') &&
          !element.closest('pre, code') &&
          element.scrollWidth > element.clientWidth + 2,
      )
      .map(
        (element) =>
          `${element.tagName.toLowerCase()}: ${element.textContent?.trim().slice(0, 100)}`,
      );

    const clippedLabels: string[] = [];
    for (const button of root.querySelectorAll('button')) {
      if (!button.getClientRects().length) continue;
      const bounds = button.getBoundingClientRect();
      const textNodes = document.createTreeWalker(button, NodeFilter.SHOW_TEXT);

      let text: Node | null;
      while ((text = textNodes.nextNode())) {
        if (!text.textContent?.trim()) continue;
        const range = document.createRange();
        range.selectNodeContents(text);
        if (
          Array.from(range.getClientRects()).some(
            (rect) =>
              rect.width > 0 &&
              rect.height > 0 &&
              (rect.left < bounds.left - 1 ||
                rect.right > bounds.right + 1 ||
                rect.top < bounds.top - 1 ||
                rect.bottom > bounds.bottom + 1),
          )
        )
          clippedLabels.push(text.textContent.trim());
      }
    }
    return {
      overlaps,
      overflowingRegions,
      clippedLabels,
      hasOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    };
  });
  expect(layout).toEqual({
    overlaps: [],
    overflowingRegions: [],
    clippedLabels: [],
    hasOverflow: false,
  });
};
