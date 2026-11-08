import { expect, type Locator } from '@playwright/test';

export const assertAccountReflow = async (region: Locator) => {
  const layout = await region.evaluate((root) => {
    const elements = Array.from(
      root.querySelectorAll<HTMLElement>('a[href], label[for], input, button, [role="alert"]'),
    ).filter((element) => element.getClientRects().length);
    const getVisibleControlBounds = (element: HTMLElement) => {
      const bounds = element.getBoundingClientRect();
      const visible = {
        left: bounds.left,
        right: bounds.right,
        top: bounds.top,
        bottom: bounds.bottom,
      };

      // Visibility toggles and number steppers occupy the input's reserved padding.
      if (element.tagName === 'INPUT') {
        const style = getComputedStyle(element);
        visible.left += parseFloat(style.paddingLeft);
        visible.right -= parseFloat(style.paddingRight);
      }

      for (let ancestor = element.parentElement; ancestor; ancestor = ancestor.parentElement) {
        const style = getComputedStyle(ancestor);
        const clip = ancestor.getBoundingClientRect();

        if (style.overflowX !== 'visible') {
          visible.left = Math.max(visible.left, clip.left + ancestor.clientLeft);
          visible.right = Math.min(
            visible.right,
            clip.left + ancestor.clientLeft + ancestor.clientWidth,
          );
        }

        if (style.overflowY !== 'visible') {
          visible.top = Math.max(visible.top, clip.top + ancestor.clientTop);
          visible.bottom = Math.min(
            visible.bottom,
            clip.top + ancestor.clientTop + ancestor.clientHeight,
          );
        }
      }

      return visible;
    };
    const overlaps: string[] = [];
    for (let leftIndex = 0; leftIndex < elements.length; leftIndex++)
      for (let rightIndex = leftIndex + 1; rightIndex < elements.length; rightIndex++) {
        const leftElement = elements[leftIndex],
          rightElement = elements[rightIndex];
        if (leftElement.contains(rightElement) || rightElement.contains(leftElement)) continue;
        const leftBounds = getVisibleControlBounds(leftElement),
          rightBounds = getVisibleControlBounds(rightElement);
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
          !element.matches('input, textarea, svg, svg *') &&
          getComputedStyle(element).clip !== 'rect(0px, 0px, 0px, 0px)' &&
          getComputedStyle(element).clipPath !== 'inset(50%)' &&
          !element.closest('pre, code') &&
          !(
            element.matches('[role="region"][tabindex="0"][aria-label]') &&
            ['auto', 'scroll'].includes(getComputedStyle(element).overflowX)
          ) &&
          element.scrollWidth > element.clientWidth + 2,
      )
      .map(
        (element) =>
          `${element.tagName.toLowerCase()}[${element.className}]: ${element.scrollWidth}/${element.clientWidth} ${element.textContent?.trim().slice(0, 100)}`,
      );

    const clippedLabels: string[] = [];
    for (const button of root.querySelectorAll('button')) {
      if (!button.getClientRects().length) continue;

      const bounds = button.getBoundingClientRect();
      const textNodes = document.createTreeWalker(button, NodeFilter.SHOW_TEXT);

      let text: Node | null;
      while ((text = textNodes.nextNode())) {
        if (!text.textContent?.trim()) continue;

        let ancestor = text.parentElement;
        let isVisible = true;

        while (ancestor) {
          const style = getComputedStyle(ancestor);

          if (style.visibility === 'hidden' || parseFloat(style.opacity) === 0) isVisible = false;

          if (ancestor === button) break;

          ancestor = ancestor.parentElement;
        }

        if (!isVisible) continue;

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
          clippedLabels.push(
            `${text.textContent.trim()} (${bounds.width}x${bounds.height}; ${getComputedStyle(button).fontSize}; ${button.className})`,
          );
      }
    }

    const splitHeaderWords: string[] = [];
    for (const header of root.querySelectorAll('th')) {
      const walker = document.createTreeWalker(header, NodeFilter.SHOW_TEXT);
      let text: Node | null;
      while ((text = walker.nextNode())) {
        for (const word of text.textContent!.matchAll(/\p{L}{3,}/gu)) {
          const range = document.createRange();
          range.setStart(text, word.index);
          range.setEnd(text, word.index + word[0].length);
          if (new Set(Array.from(range.getClientRects()).map((rect) => rect.top)).size > 1)
            splitHeaderWords.push(word[0]);
        }
      }
    }

    const clippedInputs = Array.from(root.querySelectorAll<HTMLInputElement>('input'))
      .filter((input) => {
        const style = getComputedStyle(input);
        return (
          !['hidden', 'checkbox', 'radio', 'range', 'color'].includes(input.type) &&
          input.getClientRects().length &&
          input.clientWidth > 0 &&
          style.visibility !== 'hidden' &&
          style.clip !== 'rect(0px, 0px, 0px, 0px)' &&
          style.clipPath !== 'inset(50%)' &&
          input.clientHeight < parseFloat(style.fontSize)
        );
      })
      .map((input) => input.getAttribute('aria-label') ?? input.type);

    const compressedLabels = Array.from(root.querySelectorAll('label'))
      .filter((label) => {
        if (!label.getClientRects().length || !label.textContent?.trim()) return false;
        const style = getComputedStyle(label);
        return parseFloat(style.lineHeight) < parseFloat(style.fontSize);
      })
      .map((label) => label.textContent?.trim());

    const overflowingPageElements =
      document.documentElement.scrollWidth > document.documentElement.clientWidth
        ? Array.from(document.body.querySelectorAll<HTMLElement>('*'))
            .filter(
              (element) =>
                element.getClientRects().length &&
                element.getBoundingClientRect().right > innerWidth + 2,
            )
            .slice(0, 12)
            .map(
              (element) =>
                `${element.tagName}: ${element.className} ${element.textContent?.slice(0, 60)}`,
            )
        : [];

    return {
      overflowingPageElements,
      splitHeaderWords,
      clippedInputs,
      compressedLabels,
      overlaps,
      overflowingRegions,
      clippedLabels,
      hasOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    };
  });
  expect.soft(layout).toEqual({
    overflowingPageElements: [],
    splitHeaderWords: [],
    clippedInputs: [],
    compressedLabels: [],
    overlaps: [],
    overflowingRegions: [],
    clippedLabels: [],
    hasOverflow: false,
  });
};
