import { TextInput } from '@mantine/core';
// @vitest-environment happy-dom
import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';

import { renderScreen } from '@/shared/lib/test/renderScreen';
import { Dialog } from '@/shared/ui/Dialog';
import { LanguageThemeControlComponent } from '@/shared/ui/LanguageThemeControlComponent';
import { WrappingSelect } from '@/shared/ui/WrappingSelect';

describe('Dialog', () => {
  it.each([false, true])(
    'closes only the active nested dialog and preserves pending restrictions with setup replay=%s',
    async (isStrictMode) => {
      const Screen = () => {
        const [isParentOpen, setIsParentOpen] = useState(false);
        const [isChildOpen, setIsChildOpen] = useState(false);
        const [isPending, setIsPending] = useState(false);

        return (
          <>
            <button onClick={() => setIsParentOpen(true)}>Open parent</button>
            <Dialog
              title="Parent editor"
              opened={isParentOpen}
              onClose={() => setIsParentOpen(false)}
            >
              <button onClick={() => setIsChildOpen(true)}>Open child</button>
              <Dialog
                title="Child editor"
                opened={isChildOpen}
                closeOnEscape={!isPending}
                onClose={() => setIsChildOpen(false)}
              >
                <button onClick={() => setIsPending(!isPending)}>Toggle pending</button>
              </Dialog>
            </Dialog>
          </>
        );
      };

      const { user } = renderScreen(<Screen />, { isStrictMode });
      const invoker = screen.getByRole('button', { name: 'Open parent' });

      await user.click(invoker);
      const childInvoker = screen.getByRole('button', { name: 'Open child' });
      await user.click(childInvoker);
      const pendingButton = screen.getByRole('button', { name: 'Toggle pending' });
      await user.click(pendingButton);
      await user.keyboard('{Escape}');

      expect(screen.getByRole('dialog', { name: 'Child editor' })).toBeTruthy();
      expect(screen.getByRole('dialog', { name: 'Parent editor' })).toBeTruthy();

      await user.click(pendingButton);
      await user.keyboard('{Escape}');

      expect(screen.queryByRole('dialog', { name: 'Child editor' })).toBeNull();
      expect(document.activeElement).toBe(childInvoker);

      await user.keyboard('{Escape}');

      expect(screen.queryByRole('dialog')).toBeNull();
      expect(document.activeElement).toBe(invoker);
    },
  );

  it.each(['menu', 'combobox'] as const)(
    'dismisses a nested %s before closing the editor',
    async (popup) => {
      const Screen = () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
          <Dialog title="Editor" opened={isOpen} onClose={() => setIsOpen(false)}>
            <LanguageThemeControlComponent />
            <WrappingSelect
              aria-label="Choice"
              data={[{ value: 'first', label: 'First' }]}
              value="first"
              onChange={() => {}}
            />
          </Dialog>
        );
      };
      const { user } = renderScreen(<Screen />);
      const target =
        popup === 'menu'
          ? screen.getByRole('button', { name: /Language:/ })
          : screen.getByRole('combobox', { name: 'Choice' });

      await user.click(target);
      await screen.findByRole(popup === 'menu' ? 'menu' : 'listbox');

      await user.keyboard('{Escape}');

      expect(target.getAttribute('aria-expanded')).toBe('false');
      expect(screen.getByRole('dialog', { name: 'Editor' })).toBeTruthy();
      expect(document.activeElement).toBe(target);

      await user.keyboard('{Escape}');

      expect(screen.queryByRole('dialog')).toBeNull();
    },
  );

  it.each([false, true])(
    'returns focus after an autofocused dialog closes with setup replay=%s',
    async (isStrictMode) => {
      const Screen = () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
          <>
            <button onClick={() => setIsOpen(true)}>Open editor</button>
            <Dialog title="Editor" opened={isOpen} onClose={() => setIsOpen(false)}>
              <TextInput label="Draft" autoFocus data-autofocus />
              <button onClick={() => setIsOpen(false)}>Save draft</button>
            </Dialog>
          </>
        );
      };

      const { user } = renderScreen(<Screen />, { isStrictMode });
      const invoker = screen.getByRole('button', { name: 'Open editor' });

      await user.click(invoker);

      expect(document.activeElement).toBe(screen.getByRole('textbox', { name: 'Draft' }));

      await user.click(
        within(screen.getByRole('dialog', { name: 'Editor' })).getByRole('button', {
          name: 'Save draft',
        }),
      );

      expect(screen.queryByRole('dialog')).toBeNull();
      expect(document.activeElement).toBe(invoker);
    },
  );
});
