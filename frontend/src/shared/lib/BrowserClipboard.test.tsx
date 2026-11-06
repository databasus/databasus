// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, onTestFinished, vi } from 'vitest';

import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import {
  stubClipboardCommand,
  stubClipboardUnavailable,
  stubClipboardWrite,
} from '@/shared/lib/test/stubClipboard';

describe('BrowserClipboard', () => {
  beforeEach(() => {
    const existingElements = new Set(document.querySelectorAll('textarea'));
    onTestFinished(() => {
      for (const element of document.querySelectorAll('textarea')) {
        if (!existingElements.has(element)) element.remove();
      }
    });
  });
  it('rejects and removes its temporary element when fallback copying is refused', async () => {
    renderScreen(<div />);
    stubClipboardUnavailable();
    stubClipboardCommand(vi.fn().mockReturnValue(false));

    await expect(BrowserClipboard.copyToClipboard('diagnostic')).rejects.toThrow();

    expect(document.querySelector('textarea')).toBeNull();
  });

  it('removes its temporary element when fallback copying throws', async () => {
    renderScreen(<div />);
    stubClipboardUnavailable();
    stubClipboardCommand(
      vi.fn(() => {
        throw new Error('clipboard denied');
      }),
    );

    await expect(BrowserClipboard.copyToClipboard('diagnostic')).rejects.toThrow(
      'clipboard denied',
    );

    expect(document.querySelector('textarea')).toBeNull();
  });

  it('copies through the native clipboard when it is available', async () => {
    renderScreen(<div />);
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboardWrite(writeText);

    await BrowserClipboard.copyToClipboard('diagnostic');

    expect(writeText).toHaveBeenCalledWith('diagnostic');
  });

  it('copies through the fallback and removes its element when it succeeds', async () => {
    renderScreen(<div />);
    stubClipboardUnavailable();
    const copyCommand = vi.fn().mockReturnValue(true);
    stubClipboardCommand(copyCommand);

    await BrowserClipboard.copyToClipboard('diagnostic');

    expect(copyCommand).toHaveBeenCalledWith('copy');
    expect(document.querySelector('textarea')).toBeNull();
  });
});
