// @vitest-environment happy-dom
import { act, screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, onTestFinished, vi } from 'vitest';

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

  it('keeps the action locked with a countdown until its delay passes', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    const onConfirm = vi.fn();
    const { user } = renderScreen(
      <ConfirmationComponent
        description="Enable checks?"
        actionText="Enable"
        isDestructive={false}
        actionDelaySeconds={10}
        onConfirm={onConfirm}
        onDecline={vi.fn()}
      />,
    );
    const dialog = within(screen.getByRole('dialog'));

    const lockedAction = dialog.getByRole('button', { name: 'Enable (10)' }) as HTMLButtonElement;
    expect(lockedAction.disabled).toBe(true);

    act(() => vi.advanceTimersByTime(9_000));

    const lastSecondAction = dialog.getByRole('button', {
      name: 'Enable (1)',
    }) as HTMLButtonElement;
    expect(lastSecondAction.disabled).toBe(true);

    act(() => vi.advanceTimersByTime(1_000));
    const unlockedAction = dialog.getByRole('button', { name: 'Enable' }) as HTMLButtonElement;
    await user.click(unlockedAction);

    expect(unlockedAction.disabled).toBe(false);
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });
});
