// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useScopedSelection } from '@/shared/hooks/useScopedSelection';
import { renderScreen } from '@/shared/lib/test/renderScreen';

afterEach(() => {
  vi.restoreAllMocks();
});

interface Props {
  scopeKey: string;
  itemIds: string[];
  isReady?: boolean;
  isMobile?: boolean;
}
function SelectionScreen({ scopeKey, itemIds, isReady = true, isMobile = false }: Props) {
  const selection = useScopedSelection({ scopeKey, itemIds, isReady, isMobile });

  return (
    <>
      <output aria-label="Selection">{selection.selectedItemId ?? 'list'}</output>
      {itemIds.map((itemId) => (
        <button key={itemId} onClick={() => selection.select(itemId)}>
          {itemId}
        </button>
      ))}
      <button onClick={() => selection.select(undefined)}>Back</button>
      <button
        onClick={() => selection.selectedItemId && selection.remove(selection.selectedItemId)}
      >
        Remove
      </button>
    </>
  );
}

describe('Scoped resource selection', () => {
  it('reads the new preference before writing when its key changes without remount', async () => {
    localStorage.setItem('scope-a', 'a2');
    localStorage.setItem('scope-b', 'b2');

    const view = renderScreen(<SelectionScreen scopeKey="scope-a" itemIds={['a1', 'a2']} />);

    expect(screen.getByLabelText('Selection').textContent).toBe('a2');

    view.rerender(<SelectionScreen scopeKey="scope-b" itemIds={['b1', 'b2']} />);

    expect(screen.getByLabelText('Selection').textContent).toBe('b2');
    expect(localStorage.getItem('scope-a')).toBe('a2');
    expect(localStorage.getItem('scope-b')).toBe('b2');
  });

  it('preserves a preference when the list is not ready', () => {
    localStorage.setItem('scope', 'removed');

    const view = renderScreen(<SelectionScreen scopeKey="scope" itemIds={[]} isReady={false} />);

    expect(localStorage.getItem('scope')).toBe('removed');

    view.rerender(<SelectionScreen scopeKey="scope" itemIds={['first', 'second']} />);

    expect(screen.getByLabelText('Selection').textContent).toBe('first');

    view.rerender(<SelectionScreen scopeKey="scope" itemIds={['second']} />);

    expect(screen.getByLabelText('Selection').textContent).toBe('second');
  });

  it('starts on the mobile list and returns to it after explicit back', async () => {
    localStorage.setItem('scope', 'first');

    const view = renderScreen(
      <SelectionScreen scopeKey="scope" itemIds={['first', 'second']} isMobile />,
    );

    expect(screen.getByLabelText('Selection').textContent).toBe('list');

    await view.user.click(screen.getByRole('button', { name: 'second' }));

    await view.user.click(screen.getByRole('button', { name: 'Remove' }));

    expect(screen.getByLabelText('Selection').textContent).toBe('first');

    await view.user.click(screen.getByRole('button', { name: 'Back' }));

    expect(screen.getByLabelText('Selection').textContent).toBe('list');
  });

  it('keeps selection usable when browser storage is unavailable', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('Unavailable');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('Unavailable');
    });
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => {
      throw new Error('Unavailable');
    });

    const view = renderScreen(<SelectionScreen scopeKey="scope" itemIds={['first', 'second']} />);

    await view.user.click(screen.getByRole('button', { name: 'second' }));

    expect(screen.getByLabelText('Selection').textContent).toBe('second');
  });
});
