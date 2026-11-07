// @vitest-environment happy-dom
import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { renderScreen } from '@/shared/lib/test/renderScreen';
import { ConfirmationComponent } from '@/shared/ui/ConfirmationComponent';

describe('ConfirmationComponent', () => {
  it.each([false, true])(
    'returns focus after its owner immediately unmounts it with setup replay=%s',
    async (isStrictMode) => {
      const Screen = () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
          <>
            <button onClick={() => setIsOpen(true)}>Open confirmation</button>
            {isOpen && (
              <ConfirmationComponent
                description="Remove channel?"
                actionText="Remove"
                isDestructive
                onConfirm={() => setIsOpen(false)}
                onDecline={() => setIsOpen(false)}
              />
            )}
          </>
        );
      };
      const { user } = renderScreen(<Screen />, { isStrictMode });
      const invoker = screen.getByRole('button', { name: 'Open confirmation' });

      await user.click(invoker);
      await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }));

      expect(screen.queryByRole('dialog')).toBeNull();
      expect(document.activeElement).toBe(invoker);
    },
  );

  it('cancels without confirming when the action is destructive', async () => {
    const onConfirm = vi.fn();
    const onDecline = vi.fn();
    const { user } = renderScreen(
      <ConfirmationComponent
        description="Remove channel?"
        actionText="Remove"
        isDestructive
        onConfirm={onConfirm}
        onDecline={onDecline}
      />,
    );
    const dialog = within(screen.getByRole('dialog'));

    await user.click(dialog.getByRole('button', { name: 'Cancel' }));

    expect(onDecline).toHaveBeenCalledTimes(1);
    expect(onConfirm).not.toHaveBeenCalled();

    await user.click(dialog.getByRole('button', { name: 'Remove' }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(onDecline).toHaveBeenCalledTimes(1);
  });
});
